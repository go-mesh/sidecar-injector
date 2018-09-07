# sidecar-injector  

To Inject sidecar following are the annotation needs to be added to client and server yaml

## Required

```
sidecar.mesher.io/inject:

example:

sidecar.mesher.io/inject: "yes"

Allowed values are

"yes" or "y"
```

## Optional

```
sidecar.mesher.io/discoveryType:

eample:

sidecar.mesher.io/discoveryType: "sc"

The allowed values are
1. sc
2. pilot
```

## Annotation required only for server(provider)

```
sidecar.mesher.io/servicePorts:

example:

sidecar.mesher.io/servicePorts: rest:9999

Where

9999 ----> Port where server(provider) is running
rest ----> This is the protocol(it can be rest or grpc)
```

## NOTE
consumer need to use http://provider_name:provider_port/ to access provider, instead of http://provider_ip:provider_port/.

if you choose to use annotation `sidecar.mesher.io/servicePorts`, then you can simply use http://provider_name/ to access provider

