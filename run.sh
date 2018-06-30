#!/bin/bash
go get -v -x -gcflags "-N -l" && chao 2>&1 | tee log.txt
