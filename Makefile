build:
	go build -o ship cmd/main.go

run:
	sudo ./ship child

cleantmp:
	@ROOTFS=tmp/rootfs; \
	if [ ! -d "$$ROOTFS" ]; then \
		echo "No rootfs found at $$ROOTFS"; \
		exit 0; \
	fi; \
	for dir in dev sys proc; do \
		if mountpoint -q "$$ROOTFS/$$dir"; then \
			echo "Unmounting $$dir..."; \
			sudo umount "$$ROOTFS/$$dir"; \
		fi \
	done; \
	echo "Deleting $$ROOTFS..."; \
	sudo rm -rf "$$ROOTFS"; \
	echo "Cleanup complete."

clean:
	rm -rf ship