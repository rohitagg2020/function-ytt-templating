# Render Example

Crossplane has introduced render functionality for quick feedback loop system. Now, we dont hae to package the function to see its behavior.
In this example, we will see the function behavior by running it locally.

## Pre-requisite
* Docker

```shell
$ cd ../../
$ go run . --insecure --debug
$ cd example/render
$ crossplane beta render xr.yaml composition.yaml functions-dev.yaml
```