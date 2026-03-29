build:
	./scripts/build.sh

build_image:
	./scripts/build_image.sh

gen:
	./scripts/gen.sh

test:
	go test ./...

setup-faiss:
	sudo ./scripts/setup_faiss.sh
