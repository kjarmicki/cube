## Cube

This repository contains my interactive notes from the book [Build an Orchestrator in Go (From Scratch) by Tim Boring](https://www.manning.com/books/build-an-orchestrator-in-go-from-scratch).

### Workarounds

error:
```
Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
```
fix:
```
sudo ln -s "$HOME/.docker/run/docker.sock" /var/run/docker.sock
```