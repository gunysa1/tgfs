.PHONY: build test install clean

build:
	/home/guny/go/bin/go build -o bin/tgfsd ./cmd/tgfsd
	/home/guny/go/bin/go build -o bin/tgfs ./cmd/tgfs

test:
	/home/guny/go/bin/go test ./... -v

install: build
	install -m 755 bin/tgfsd /usr/local/bin/tgfsd
	install -m 755 bin/tgfs /usr/local/bin/tgfs
	mkdir -p /etc/tgfs /var/lib/tgfs /mnt/tgfs
	cp -n config.yaml.example /etc/tgfs/config.yaml || true
	install -m 644 tgfs.service /etc/systemd/system/tgfs.service
	systemctl daemon-reload

clean:
	rm -rf bin/
