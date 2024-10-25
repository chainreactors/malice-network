
## Summary
This directory contains an example console application containing:
- Two different command menus
- Each with their own prompt engine
- The first menu contains the complete command set from the [reeflective/flags/example](https://github.com/reeflective/flags/tree/main/example) CLI.
- (The other one is empty)
- Special interrupt handlers to switch back and forth between menus.

## Installing
Assuming that you have a working Go toolchain:
```bash
go install github.com/reeflective/console/example
```
<!-- ```bash -->
<!-- # Clone the repository and go to example directory -->
<!-- git clone https://github.com/reeflective/console && cd console/example -->
<!---->
<!-- # Build and run the console -->
<!-- go build . && ./example -->
<!-- ``` -->

## Directories and files
The files/directories below are listed in the order in which a user would want to 
read them to fully understand how to use the various features of this library.
Note that these files are also the ones used as demonstration snippets in the [wiki](https://github.com/reeflective/console/wiki).

- `main.go`         - The entrypoint where all our bindings functions are called, and the application is run.
- `menu.go`         - In here, we create a new menu, and bind some various stuff to it.
- `commands.go`     - Here we generate and bind our cobra command tree to one of the menus.
- `interrupt.go`    - Declares some special interrupt handlers to be used on certain keystrokes.
- `.example-history` - A history file used as a source of command history, embedded in the binary.


