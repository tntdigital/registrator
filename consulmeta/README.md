# Consul Service Catalog with metadata support
Same as [Consul Service Catalog](#consul-service-catalog-recommended), but then with [metadata](#single-service-with-metadata) [support](#multiple-services-with-metadata).

To use the Consul service catalog, specify a Consul URI without a path. If no host is provided, `127.0.0.1:8500` is used. Examples:

```
$ registrator consulmeta://10.0.0.1:8500/<registry-uri-path>
```

Metadata is stored as `<registry-uri-path>/<service-name>/<key> = <value>`
