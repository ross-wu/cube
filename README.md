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
$ go build server.go
```

#### Usage:

**Simple usage**

```
$ ./server
```

then in brower:
http://localhost/cube?U=wwwwwwwww&L=ggggggggg&F=bbbbbbbbb&R=ooooooooo&B=rrrrrrrrr&D=yyyyyyyyy


More examples:

  * http://localhost/cube?U=wwwwwwwww&L=ggggggggg&F=bbbbbbbbb&R=ooooooooo&B=rrrrrrrrr&D=yyyyyyyyy
  * http://localhost/cube?U=bwwbyryyr&L=wrrrgyyoo&F=ggboobygy&R=wygwbyoro&B=roobrgwwb&D=bogbwwrgg
  * http://localhost/cube?U=yyoyygbwo&L=ggwooboob&F=rrwybwyoo&R=brgbrgyrg&B=wrrwgywoy&D=rbbgwbgwr

**Set http port**

```
$ ./server --port=8080
```

**Step-by-step**

```
$ ./server -v
```

**Step-by-step w/ debug info**

```
$ ./server --debug
```

