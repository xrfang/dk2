BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
HASH=$(shell git log -n1 --pretty=format:%h)
REVS=$(shell git log --oneline|wc -l)
native: release
arm: export GOOS=linux
arm: export GOARCH=arm
arm: export GOARM=7
arm: release
upx:
	upx -9 dk*
debug: setver geneh compdbg
release: setver geneh comprel upx
geneh: #generate error handler
	@for tpl in `find . -type f |grep errors.tpl`; do \
        target=`echo $$tpl|sed 's/\.tpl/\.go/'`; \
        pkg=`basename $$(dirname $$tpl)`; \
        sed "s/package main/package $$pkg/" errors.go > $$target; \
		sed -i "s/PKGNAME/$$pkg/" $$target; \
    done
setver:
	cp verinfo.tpl version.go
	sed -i 's/{_BRANCH}/$(BRANCH)/' version.go
	sed -i 's/{_G_HASH}/$(HASH)/' version.go
	sed -i 's/{_G_REVS}/$(REVS)/' version.go
comprel:
	go build -ldflags="-s -w" .
compdbg:
	go build -race -gcflags=all=-d=checkptr=0 .
clean:
	rm -fr version.go dk
