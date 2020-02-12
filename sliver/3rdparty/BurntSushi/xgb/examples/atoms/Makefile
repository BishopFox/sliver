atoms:
	go build

test: atoms
	./atoms --requests 500000 --cpu 1 \
		--cpuprof cpu1.prof --memprof mem1.prof > atoms1.info 2>&1
	./atoms --requests 500000 --cpu 2 \
		--cpuprof cpu2.prof --memprof mem2.prof > atoms2.info 2>&1
	./atoms --requests 500000 --cpu 3 \
		--cpuprof cpu3.prof --memprof mem3.prof > atoms3.info 2>&1
	./atoms --requests 500000 --cpu 6 \
		--cpuprof cpu6.prof --memprof mem6.prof > atoms6.info 2>&1
