binemu
----

This is a fork of [lunixbochs/usercorn](https://github.com/lunixbochs/usercorn).
This repository is more an experimental branch that wants to focus on completely
emulate the binary with a virutal kernel that does not interact with the host
kernel.

# Build

```
make deps
make protos
make
```

# Run

To run the binary in a simple "unbunuish" environment:

`./binemu --config_path=bins/ubuntu64/ubuntu.textproto PATH_TO_BIN [ARGS...]`
