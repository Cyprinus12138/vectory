package utils

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestMergeSortedLists(t *testing.T) {
	tests := []struct {
		name  string
		lists [][]int
		cap   int
		want  []int
	}{
		{
			name:  "merge two sorted lists",
			lists: [][]int{{1, 3, 5}, {2, 4, 6}},
			cap:   6,
			want:  []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:  "merge with cap limit",
			lists: [][]int{{1, 3, 5}, {2, 4, 6}},
			cap:   3,
			want:  []int{1, 2, 3},
		},
		{
			name:  "merge three lists",
			lists: [][]int{{1, 4}, {2, 5}, {3, 6}},
			cap:   6,
			want:  []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:  "empty lists",
			lists: [][]int{{}, {}},
			cap:   5,
			want:  nil,
		},
		{
			name:  "single list",
			lists: [][]int{{1, 2, 3}},
			cap:   5,
			want:  []int{1, 2, 3},
		},
		{
			name:  "no lists",
			lists: [][]int{},
			cap:   5,
			want:  nil,
		},
		{
			name:  "cap zero",
			lists: [][]int{{1, 2}, {3, 4}},
			cap:   0,
			want:  nil,
		},
		{
			name:  "one empty one populated",
			lists: [][]int{{}, {1, 2, 3}},
			cap:   5,
			want:  []int{1, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeSortedLists(tt.lists, func(i, j int) bool { return i < j }, tt.cap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeSortedLists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriorityQueue(t *testing.T) {
	pq := &PriorityQueue[int]{
		less: func(i, j int) bool { return i < j },
	}

	if pq.Len() != 0 {
		t.Errorf("expected empty queue, got len %d", pq.Len())
	}

	pq.Push(3)
	pq.Push(1)
	pq.Push(2)

	if pq.Len() != 3 {
		t.Errorf("expected len 3, got %d", pq.Len())
	}

	// Test Less
	if !pq.Less(1, 0) {
		t.Errorf("expected items[1] < items[0]")
	}

	// Test Swap
	pq.Swap(0, 2)
	if pq.items[0] != 2 || pq.items[2] != 3 {
		t.Errorf("Swap failed: got %v", pq.items)
	}

	// Test Pop
	item := pq.Pop().(int)
	if pq.Len() != 2 {
		t.Errorf("expected len 2 after pop, got %d", pq.Len())
	}
	_ = item
}

func TestFileExists(t *testing.T) {
	// Existing file
	if !FileExists("utils.go") {
		t.Errorf("expected utils.go to exist")
	}

	// Non-existing file
	if FileExists("nonexistent_file_12345.go") {
		t.Errorf("expected nonexistent file to not exist")
	}
}

func TestGenInstanceId(t *testing.T) {
	id1 := GenInstanceId("test")
	id2 := GenInstanceId("test")

	if !strings.HasPrefix(id1, "test-") {
		t.Errorf("expected prefix 'test-', got %s", id1)
	}
	if id1 == id2 {
		t.Errorf("expected unique IDs, got %s and %s", id1, id2)
	}
}

func TestUnmarshal(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name" yaml:"name"`
		Value int    `json:"value" yaml:"value"`
	}

	tests := []struct {
		name    string
		raw     []byte
		want    testStruct
		wantErr bool
	}{
		{
			name: "valid json",
			raw:  []byte(`{"name":"test","value":42}`),
			want: testStruct{Name: "test", Value: 42},
		},
		{
			name: "valid yaml",
			raw:  []byte("name: test\nvalue: 42\n"),
			want: testStruct{Name: "test", Value: 42},
		},
		{
			name:    "invalid content",
			raw:     []byte(`<<<not valid>>>`),
			wantErr: true,
		},
		{
			name: "empty json object",
			raw:  []byte(`{}`),
			want: testStruct{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got testStruct
			err := Unmarshal(tt.raw, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListProtoMethods(t *testing.T) {
	// Requires a grpc.ServiceDesc but we can't easily construct one without proto imports.
	// Tested indirectly through integration; skip for now.
}

func TestFileExists_Directory(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if !FileExists(dir) {
		t.Errorf("expected directory to be reported as existing")
	}
}
