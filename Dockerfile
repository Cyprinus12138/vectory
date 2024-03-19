FROM kitware/cmake:ci-debian12-x86_64-2024-03-04 AS faiss_builder

RUN \
  apt update && \
  apt install -y cmake && \
  git clone https://github.com/facebookresearch/faiss.git && \
  cd faiss && \
  cmake -B build -DFAISS_ENABLE_GPU=OFF -DFAISS_ENABLE_C_API=ON -DBUILD_SHARED_LIBS=ON . && \
  make -C build && \
  make -C build install


FROM golang:1.20

COPY --from=faiss_builder faiss/build/c_api/libfaiss_c.so /usr/lib

ARG GIT_TOKEN=""
WORKDIR /app
COPY . ./
RUN make build
ENTRYPOINT ["./bin/server/main"]
