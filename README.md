# bedrockprune

## Goal

A simple Go application that will let me prune unwanted chunks from my
Minecraft Bedrock save files.

## Current status

- [x] Reading leveldb files and finding block names
- [x] Initial prototype of Gioui GUI ([./cmd/maze](./cmd/maze))
- [ ] Tileserver to avoid 200K images at far zoom - WIP
- [ ] Downloading textures
- [ ] Mapping names to textures
- [ ] Tying GUI to actual data
- [ ] Pruning
- [ ] Rectangle selection for pruning
- [ ] Everything else...

## Safety

I'm testing it on my save files. I **DO NOT** expect it to be generally
safe. Use at your own risk.

## Version support

**Horribly lacking.** If it didn't show up in my backup, I don't care
about it. If you find something that doesn't work, and send me your
backup, I'll try to make it work.

## Acknowledgements

The giants upon whose shoulders I'm standing:

- [Dragonfly Go bedrock server](https://github.com/df-mc/dragonfly) -
  leveldb parsing, use of constants and structs, extensive reading of
  their code to understand how things work. My leveldb reading code is
  basically just a pared down translation of their code.
- [gophertunnel](https://github.com/Sandertv/gophertunnel) - NBT parsing
- [Gio UI](https://gioui.org) - Cross-platform GUI for GO
