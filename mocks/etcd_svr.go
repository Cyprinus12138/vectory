// Copyright 2018 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mocks

import (
	"context"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
)

// MockServer provides a mocked out grpc server of the etcdserver interface.
type MockServer struct {
	ln         net.Listener
	Network    string
	Address    string
	GRPCServer *grpc.Server
}

func (ms *MockServer) ResolverAddress() resolver.Address {
	switch ms.Network {
	case "unix":
		return resolver.Address{Addr: fmt.Sprintf("unix://%s", ms.Address)}
	case "tcp":
		return resolver.Address{Addr: ms.Address}
	default:
		panic("illegal network type: " + ms.Network)
	}
}

// MockServers provides a cluster of mocket out gprc servers of the etcdserver interface.
type MockServers struct {
	mu      sync.RWMutex
	Servers []*MockServer
	wg      sync.WaitGroup
}

// StartMockServers creates the desired count of mock servers
// and starts them.
func StartMockServers(count int) (ms *MockServers, err error) {
	return StartMockServersOnNetwork(count, "tcp")
}

// StartMockServersOnNetwork creates mock servers on either 'tcp' or 'unix' sockets.
func StartMockServersOnNetwork(count int, network string) (ms *MockServers, err error) {
	switch network {
	case "tcp":
		return startMockServersTCP(count)
	case "unix":
		return startMockServersUnix(count)
	default:
		return nil, fmt.Errorf("unsupported network type: %s", network)
	}
}

func startMockServersTCP(count int) (ms *MockServers, err error) {
	addrs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		addrs = append(addrs, "localhost:0")
	}
	return startMockServers("tcp", addrs)
}

func startMockServersUnix(count int) (ms *MockServers, err error) {
	dir := os.TempDir()
	addrs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		f, err := os.CreateTemp(dir, "etcd-unix-so-")
		if err != nil {
			return nil, fmt.Errorf("failed to allocate temp file for unix socket: %v", err)
		}
		fn := f.Name()
		err = os.Remove(fn)
		if err != nil {
			return nil, fmt.Errorf("failed to remove temp file before creating unix socket: %v", err)
		}
		addrs = append(addrs, fn)
	}
	return startMockServers("unix", addrs)
}

func startMockServers(network string, addrs []string) (ms *MockServers, err error) {
	ms = &MockServers{
		Servers: make([]*MockServer, len(addrs)),
		wg:      sync.WaitGroup{},
	}
	defer func() {
		if err != nil {
			ms.Stop()
		}
	}()
	for idx, addr := range addrs {
		ln, err := net.Listen(network, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen %v", err)
		}
		ms.Servers[idx] = &MockServer{ln: ln, Network: network, Address: ln.Addr().String()}
		ms.StartAt(idx)
	}
	return ms, nil
}

// StartAt restarts mock server at given index.
func (ms *MockServers) StartAt(idx int) (err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.Servers[idx].ln == nil {
		ms.Servers[idx].ln, err = net.Listen(ms.Servers[idx].Network, ms.Servers[idx].Address)
		if err != nil {
			return fmt.Errorf("failed to listen %v", err)
		}
	}

	svr := grpc.NewServer()
	pb.RegisterKVServer(svr, newMockKVServer())
	pb.RegisterLeaseServer(svr, &mockLeaseServer{})
	ms.Servers[idx].GRPCServer = svr

	ms.wg.Add(1)
	go func(svr *grpc.Server, l net.Listener) {
		svr.Serve(l)
	}(ms.Servers[idx].GRPCServer, ms.Servers[idx].ln)
	return nil
}

// StopAt stops mock server at given index.
func (ms *MockServers) StopAt(idx int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.Servers[idx].ln == nil {
		return
	}

	ms.Servers[idx].GRPCServer.Stop()
	ms.Servers[idx].GRPCServer = nil
	ms.Servers[idx].ln = nil
	ms.wg.Done()
}

// Stop stops the mock server, immediately closing all open connections and listeners.
func (ms *MockServers) Stop() {
	for idx := range ms.Servers {
		ms.StopAt(idx)
	}
	ms.wg.Wait()
}

type mockKVServer struct {
	mu       sync.Mutex
	data     map[string]string
	revision map[string]int
}

func newMockKVServer() *mockKVServer {
	return &mockKVServer{
		mu:       sync.Mutex{},
		data:     make(map[string]string),
		revision: make(map[string]int),
	}
}

func (m *mockKVServer) Range(ctx context.Context, req *pb.RangeRequest) (*pb.RangeResponse, error) {
	key := string(req.GetKey())
	if key == "" {
		return &pb.RangeResponse{}, rpctypes.ErrGRPCEmptyKey
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var kvs []*mvccpb.KeyValue

	if len(req.GetRangeEnd()) > 0 {
		for k, v := range m.data {
			if k >= string(req.GetKey()) && (req.GetRangeEnd() == nil || key < string(req.GetRangeEnd())) {
				kvs = append(kvs, &mvccpb.KeyValue{Key: []byte(key), Value: []byte(v)})
			}
		}
	} else {
		value, ok := m.data[key]
		if !ok {
			logger.Info("get key not found", logger.String("key", key), logger.Interface("data", m.data))
			return &pb.RangeResponse{}, rpctypes.ErrGRPCKeyNotFound // Key not found
		}
		kvs = append(kvs, &mvccpb.KeyValue{Key: []byte(key), Value: []byte(value)})
	}

	return &pb.RangeResponse{Kvs: kvs, Count: int64(len(kvs))}, nil
}

func (m *mockKVServer) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer func() {
		logger.Info("put key finished", logger.String("key", string(req.GetKey())), logger.Interface("data", m.data))
	}()
	if _, ok := m.revision[string(req.GetKey())]; ok {
		m.revision[string(req.GetKey())] += 1
	}

	m.data[string(req.GetKey())] = string(req.GetValue())
	return &pb.PutResponse{}, nil
}

func (m *mockKVServer) DeleteRange(ctx context.Context, req *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	key := string(req.GetKey())
	if key == "" {
		return &pb.DeleteRangeResponse{}, rpctypes.ErrGRPCEmptyKey
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var kvs []*mvccpb.KeyValue

	if len(req.GetRangeEnd()) > 0 {
		for k, v := range m.data {
			if k >= string(req.GetKey()) && (req.GetRangeEnd() == nil || key < string(req.GetRangeEnd())) {
				kvs = append(kvs, &mvccpb.KeyValue{Key: []byte(key), Value: []byte(v)})
			}
		}
	} else {
		value, ok := m.data[key]
		if !ok {
			return &pb.DeleteRangeResponse{}, rpctypes.ErrGRPCKeyNotFound // Key not found
		}
		delete(m.data, key)
		delete(m.revision, key)
		kvs = append(kvs, &mvccpb.KeyValue{Key: []byte(key), Value: []byte(value)})
	}

	return &pb.DeleteRangeResponse{PrevKvs: kvs, Deleted: int64(len(kvs))}, nil
}

func (m *mockKVServer) Txn(context.Context, *pb.TxnRequest) (*pb.TxnResponse, error) {
	return &pb.TxnResponse{}, nil
}

func (m *mockKVServer) Compact(context.Context, *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	return &pb.CompactionResponse{}, nil
}

func (m *mockKVServer) Lease(context.Context, *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error) {
	return &pb.LeaseGrantResponse{}, nil
}

type mockLeaseServer struct{}

func (s mockLeaseServer) LeaseGrant(ctx context.Context, req *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error) {
	return &pb.LeaseGrantResponse{ID: 123456, TTL: req.GetTTL()}, nil
}

func (s *mockLeaseServer) LeaseRevoke(context.Context, *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error) {
	return &pb.LeaseRevokeResponse{}, nil
}

func (s *mockLeaseServer) LeaseKeepAlive(pb.Lease_LeaseKeepAliveServer) error {
	return nil
}

func (s *mockLeaseServer) LeaseTimeToLive(context.Context, *pb.LeaseTimeToLiveRequest) (*pb.LeaseTimeToLiveResponse, error) {
	return &pb.LeaseTimeToLiveResponse{}, nil
}

func (s *mockLeaseServer) LeaseLeases(context.Context, *pb.LeaseLeasesRequest) (*pb.LeaseLeasesResponse, error) {
	return &pb.LeaseLeasesResponse{}, nil
}
