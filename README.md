# cube
Rubik's cube resolver.

## Usage

### Step 1. Build the `kociemba` cube solver tool.

`kociemba/` contains C99 implementation of Kociemba's two-phase algorithm
for solving Rubik's Cube.
It's forked from `https://github.com/muodov/kociemba`, please refer to the
original repository for more info.

Usage:

```
$ cd kociemba/
$ make
$ ./bin/kociemba DRLUUBFBRBLURRLRUBLRDDFDLFUFUFFDBRDUBRUFLLFDDBFLUBLRBD
```

### Step 2. Build the LEGO ev3 solver

```
$ go build main.go
```

#### Usage:

**Simple usage**

```
$ cat in.1 | ./main
```

**Step-by-step output**

```
$ cat demo.in | ./main -v
```

**Step-by-step w/ debug info**

```
$ cat demo.in |./main --debug
```

