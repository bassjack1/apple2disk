.PHONY: clean all

GO_APPS := bin/floppy_disk_image_file_to_serial_install

all : ${GO_APPS}

${GO_APPS} : bin/% : go/app/%.go
	go build -o $@ $<

clean :
	rm -f ${GO_APPS}
