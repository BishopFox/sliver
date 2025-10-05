# This Makefile is used by the developer. It is not needed in any way to build
# a checkout of the XGB repository.
# It will be useful, however, if you are hacking at the code generator.
# i.e., after making a change to the code generator, run 'make' in the
# xgb directory. This will build xgbgen and regenerate each sub-package.
# 'make test' will then run any appropriate tests (just tests xproto right now).
# 'make bench' will test a couple of benchmarks.
# 'make build-all' will then try to build each extension. This isn't strictly
# necessary, but it's a good idea to make sure each sub-package is a valid
# Go package.

# My path to the X protocol XML descriptions.
ifndef XPROTO
XPROTO=/usr/share/xcb
endif

# All of the XML files in my /usr/share/xcb directory EXCEPT XKB. -_-
# This is intended to build xgbgen and generate Go code for each supported
# extension.
all: build-xgbgen \
		 bigreq.xml composite.xml damage.xml dpms.xml dri2.xml \
		 ge.xml glx.xml randr.xml record.xml render.xml res.xml \
		 screensaver.xml shape.xml shm.xml xc_misc.xml \
		 xevie.xml xf86dri.xml xf86vidmode.xml xfixes.xml xinerama.xml \
		 xprint.xml xproto.xml xselinux.xml xtest.xml \
		 xvmc.xml xv.xml

build-xgbgen:
	(cd xgbgen && go build)

# Builds each individual sub-package to make sure its valid Go code.
build-all: bigreq.b composite.b damage.b dpms.b dri2.b ge.b glx.b randr.b \
					 record.b render.b res.b screensaver.b shape.b shm.b xcmisc.b \
					 xevie.b xf86dri.b xf86vidmode.b xfixes.b xinerama.b \
					 xprint.b xproto.b xselinux.b xtest.b xv.b xvmc.b

%.b:
	(cd $* ; go build)

# Installs each individual sub-package.
install: bigreq.i composite.i damage.i dpms.i dri2.i ge.i glx.i randr.i \
					 record.i render.i res.i screensaver.i shape.i shm.i xcmisc.i \
					 xevie.i xf86dri.i xf86vidmode.i xfixes.i xinerama.i \
					 xprint.i xproto.i xselinux.i xtest.i xv.i xvmc.i
	go install

%.i:
	(cd $* ; go install)

# xc_misc is special because it has an underscore.
# There's probably a way to do this better, but Makefiles aren't my strong suit.
xc_misc.xml: build-xgbgen
	mkdir -p xcmisc
	xgbgen/xgbgen --proto-path $(XPROTO) $(XPROTO)/xc_misc.xml > xcmisc/xcmisc.go

%.xml: build-xgbgen
	mkdir -p $*
	xgbgen/xgbgen --proto-path $(XPROTO) $(XPROTO)/$*.xml > $*/$*.go

# Just test the xproto core protocol for now.
test:
	(cd xproto ; go test)

# Force all xproto benchmarks to run and no tests.
bench:
	(cd xproto ; go test -run 'nomatch' -bench '.*' -cpu 1,2,3,6)

# gofmt all non-auto-generated code.
# (auto-generated code is already gofmt'd.)
# Also do a column check (80 cols) after a gofmt.
# But don't check columns on auto-generated code, since I don't care if they
# break 80 cols.
gofmt:
	gofmt -w *.go xgbgen/*.go examples/*.go examples/*/*.go xproto/xproto_test.go
	colcheck *.go xgbgen/*.go examples/*.go examples/*/*.go xproto/xproto_test.go

push:
	git push origin master
	git push github master

