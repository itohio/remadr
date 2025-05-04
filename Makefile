# List all examples by reading the directory names under ./examples
EXAMPLES := $(patsubst examples/%,%,$(wildcard examples/*))

# Flash a specific example with `make example1`
$(EXAMPLES):
	tinygo flash -target=pico ./examples/$@

.PHONY: all $(EXAMPLES)

# Serial monitor
monitor:
	tinygo monitor
