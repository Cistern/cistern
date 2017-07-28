lm2 [![CircleCI](https://circleci.com/gh/Preetam/lm2/tree/master.svg?style=svg&circle-token=6cf313cbc68be74cb6c8aebcef157cfbd05e54e3)](https://circleci.com/gh/Preetam/lm2/tree/master) [![GoDoc](https://godoc.org/github.com/Preetam/lm2?status.svg)](https://godoc.org/github.com/Preetam/lm2)
===
lm2 (listmap2) is an ordered key-value storage library.

It provides

* Ordered key-value data model
* Append-only modifications
* Fully durable, atomic writes
* Cursors with snapshot reads

Because it is append-only, records are never actually deleted.
You will have to rewrite a collection to reclaim space.

License
---
BSD (see LICENSE)
