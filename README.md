# gotalog
Gotalog is a Golang implementation of datalog, a port of MITRE corporation's Lua implementation.
Gotalog is licensed under the GPL; see LICENSE.

# Package
Gotalog comes in two parts; a port of the search strategy in `datalog.go`, and a proof-of-concept 
implementation of a number of databases with different runtime behaviors.

Outside of database constructors, the public symbols are defined in `interface.go`

# Usage

Gotalog can be interacted with either through text, or by directly constructing commands
and passing them into `Apply()`. 

We provide three database implementations: an in-memory database, a log-backed database,
and a threadsafe implementation.

The `cli` submodule has a minimal demonstration of use of the parsing API.

# Performance

In some informal tests using large datalog problems from the web (see the files in `tests/`),
gotalog's performance is better than the MITRE implementation (running using vanilla Lua, not luajit)
by around 20%. At peak, its memory consumption is several times that of the MITRE implementation.
