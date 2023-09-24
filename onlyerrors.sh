#!/bin/zsh

## Recompile With no Warning errors, Only errors

RUSTFLAGS="-A warnings" cargo test
