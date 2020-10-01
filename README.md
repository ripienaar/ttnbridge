## What?

Received HTTP pushes from The Things Network and bridge it to other locations.

At present this supports only Logging and pushing to Graphite.

## Running

The `ripienaar/ttnbridge` Docker container can be used to run it:

```nohighlight
$ docker run ripienaar/ttnbridge --help
usage: ttnbridge --listen-key=LISTEN-KEY [<flags>]

The Things Network HTTP Push bridge

Flags:
  --help                         Show context-sensitive help (also try
                                 --help-long and --help-man).
  --version                      Show application version.
  --listen-host="0.0.0.0"        Host to listen on
  --listen-port=80               Port to listen on
  --listen-key=LISTEN-KEY        Key to expect in the BridgeKey header
  --logger                       Enable the Logger processor
  --graphite                     Enable the Graphite processor
  --graphite-host=GRAPHITE-HOST  Hostname of the Graphite server
  --graphite-port=GRAPHITE-PORT  Port of the Graphite server
  --graphite-tries=10            Number of times to retry publishing to graphite
  --graphite-prefix="ttn"        Prefix used for the Graphite metrics
```

For each setting there is a matching Environment variable supported, `--graphite-tries` can be set with `GRAPHITE_TRIES`.

## Authentication?

The option `--listen-key` is required, this is a string that should be set in the `BridgeKey` Custom Header in TTN. 

## Contact?

R.I.Pienaar / [devco.net](https://www.devco.net/) / [@ripienaar](https://twitter.com/ripienaar)
