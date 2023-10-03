TARGET ?= qsr

build:
	go build -o ${TARGET} .

strip:
	strip ${TARGET}

clean:
	rm -f ${TARGET}
	
all:
	build
	strip