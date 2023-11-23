# function-ytt-templating

This composition function allows you to compose Crossplane resources using Ytt templates. 

Here's an example:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: demo-inline
spec:
  compositeTypeRef:
    apiVersion: demo.crossplane.io/v1beta1
    kind: XR
  mode: Pipeline
  pipeline:
    - step: render-templates
      functionRef:
        name: function-ytt-templating
      input:
        apiVersion: ytt.fn.crossplane.io/v1beta1
        kind: YTT
        source: Inline
        inline: |
            #@ load("@ytt:data", "data")
            ---
            apiVersion: s3.aws.upbound.io/v1beta1
            kind: Bucket
            metadata:
              annotations:
                ytt.fn.crossplane.io/composition-resource-name: bucket
            spec:
              forProvider:
                region: #@ data.values.region
```
## Using this function

This function can load templates from one source: `Inline`.

Use the `Inline` source to specify a simple template inline in your Composition.
Multiple YAML manifests can be specified using the `---` document separator.

To mark a desired composed resource as ready, use the
`ytt.fn.crossplane.io/ready` annotation:

```yaml
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
metadata:
  annotations:
    ytt.fn.crossplane.io/composition-resource-name: bucket
    ytt.fn.crossplane.io/ready: True
spec: {}
```

See the [example](example) directory for examples that you can run locally using
the Crossplane CLI:

```shell
$ crossplane beta render xr.yaml composition.yaml functions.yaml
```

## Developing this function

This function uses [Go][go], [Docker][docker], and the [Crossplane CLI][cli] to
build functions.

```shell
# Run code generation - see input/generate.go
$ go generate ./...

# Run tests - see fn_test.go
$ go test ./...

# Build the function's runtime image - see Dockerfile
$ docker build . --tag=runtime

# Build a function package - see package/crossplane.yaml
$ crossplane xpkg build -f package --embed-runtime-image=runtime
```

[go]: https://go.dev
[docker]: https://www.docker.com
[cli]: https://docs.crossplane.io/latest/cli
