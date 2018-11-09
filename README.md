# RegStat - persisting Docker registry notifications

The v2 Docker registry (also known as [Docker distribution](https://github.com/docker/distribution)) can be
configured to notify via a webhook whenever any events (push, pull, delete) occur.

This simple application is designed to act as a server for those webhooks, parsing the webhook JSON body
and then persisting details of a registry's activity into a Postgres database.

The contents of the database can then be queried to determine interesting facts about the objects in the
registry, facts that are not easy to astertain directly from the registry itself.

For instance ...

* list of images in the registry
* total number of blobs in the registry
* images that are more than X days old
* images that haven't been pulled for Y months
* orphaned blobs
* manifests that reference missing blobs
* images with the most tags

## Database schema

RegStat expects to be provided with a connection to a Postgres database. Any modern Postgres version should
do. It's been tested with 9.5 and 10.3.

On start up RegStat will attempt to create a `regstat` schema and the following tables in that schema, if
they don't already exist ...

table | columns | description
----- | ------- | -----------
blobs | digest, pushed, pulled | list of blobs in the registry 
manifests | digest, pushed, pulled | list of manifests in the registry
manifest_blob | manifest_digest, blob_digest | join table, linking manifests to their blobs
tags | name, registry, repository, tag, manifest_digest, pushed, pulled | list of tags in the registry and the manifests that they represent; name is a concatenation of registry, repository and tag
deleted_blobs | digest, pushed, pulled, deleted | list of deleted blobs in the registry 
deleted_manifests | digest, pushed, pulled, deleted | list of deleted manifests in the registry
deleted_manifest_blob | manifest_digest, blob_digest, deleted | join table, linking deleted manifests to their deleted blobs
deleted_tags | name, registry, repository, tag, manifest_digest, pushed, pulled, deleted | list of deleted tags in the registry and the deleted manifests that they represent

The `blobs`, `manifests` and `tags` tables, and the `deleted_` equivalents, all contain `pushed` and `pulled` timestamp fields, which contain the time
of the most recent push or pull event that affected that object.

The four `deleted_` tables are the same as the main tables, except for the addition of an extra `deleted` timestamp column and the
dropping of some constraints. These, fairly obviously, get populated as registry objects are deleted. They are
intended to act as an audit trail for deletion events.

## Running RegStat

### Docker

Quick and easy means to get RegStat going.

This starts a Postges database in a container and RegStat in another container. The RegStat server is listening on port 3333. Postgres is accessible as user "postgres" with no password on port 5432.

Obviously, for any semi-serious use, you'll want to use a Postgres database backed by reliable storage and likely add more options to the Regstat command line.

````
$ docker run -d --name=regstat-postgres -e POSTGRES_PASSWORD='' -p 5432:5432 postgres:10
$ docker run -d --name=regstat --net=host vleurgat/regstat:latest
````

(Note that the `--net=host` option is important. It allows the `regstat` container to connect to the `regstat-postgres` one as `localhost:5432`. There are other, better, ways to link these containers, but this works well enough for a simple example.)

You can supply extra RegStat args on the end of the second `docker run` command ...

````
$ docker run -d --name=regstat --net=host vleurgat/regstat:latest -port 4444
$ docker logs regstat
1970/01/01 22:34:45 Server now listening on :4444
````

Also, for example, adding config files via a volume and additional arguments ...

````
$ docker run -d --name=regstat --net=host \
  -v $HOME/cfgfiles:/cfgfiles \
  vleurgat/regstat:latest \
  -docker-config /cfgfiles/config.json \
  -equiv-registries /cfgfiles/equiv-registries.json
````

### Command line

````
$ regstat -h
Usage of regstat:
  -docker-config string
    	the path to the Docker registry config.json file, used to obtain login credentials
  -equiv-registries string
    	the path to the equiv-registries.json file, used to combine equivalent registries
  -pg-conn-str string
    	the Postgres connect string, e.g. "host=host port=1234 user=user password=pw ..."
  -port string
    	the port number to listen on (default "3333")
````

At a minimum RegStat takes up to four arguments ...

* the port number to listen on, provided using the `-port` option, defaults to 3333
* the Postgres connection string, provided using the `-pg-conn-str` option, defaults to "locahost:5432" as user "postgres" with no password
* registry authorization details, via a Docker `config.json` file, using the `-docker-config` option, defaults to none
* an equivalent registries file, see *Equivalent registries* below, using the `-equiv-registries` option, defaults to none

Note, be sure to quote the Postgres connection string.

A full example ...
````
$ regstat -port 9999 \
-pg-conn-str "host=localhost port=5432 user=regstat password=regstat dbname=regstat sslmode=disable" \
-docker-config $HOME/.docker/config.json \
-equiv-registries $HOME/.docker/equiv-registries.json
````

## Registry authorization

When processing the push of a Docker manifest, RegStat will make a RESTful call back to the registry to GET the
contents of that manifest. This allows RegStat to determine which blobs the manifest refers to, and so maintain
that information via the `manifest_blob` table.

If the registry requires the GET connection to be authorized then you must use the `-docker-config` option to
provide the path to a Docker `config.json` file that lists the appropriate authorization tokens for the
registry.

RegStat supports both basic and brearer/token authorization methods.

## Equivalent registries

It may be that one registry is known by different names. For instance clients may use different DNS aliases or
IP addresses, or a default (80, 443) port number may or may not always be provided.

These equivalent registries create noise in the database: one image may be present multiple times in the
`tags` table. To get around this you can create a *equivalent registries* JSON file that lists the preferred
name for registries and any effective aliases for them.

The format of the file is ... 

````
{
  "registry" : [
    "alias1",
    "alias2",
    ...
  ],
  "another_registry" : [
    "alias3",
    "alias4",
    ...
  ],
  ...
}
````

For the example above any use of `alias1` or `alias2` will be mapped to `registry`, and use of `alias3` or `alias4`
mapped to `another_registry`.

## Configuring the Docker registry to notify RegStat

See the Docker documentation: [work with notifications](https://docs.docker.com/registry/notifications/)

Currently RegStat does not require any authorization tokens and listens on a HTTP port, rather than HTTPS.

Example configuration ...

````
notifications:
  endpoints:
    - name: RegStat
      url: http://regstat.host:3333
      timeout: 500ms
      threshold: 5
      backoff: 1s
````

## Building RegStat

Linux static binary ...
````
$ go get github.com/vleurgat/regstat/cmd/regstat
$ CGO_ENABLED=0 go install -a -ldflags '-extldflags="-static"' github.com/vleurgat/regstat/cmd/regstat
````

Windows ...
````
C:\> go get github.com/vleurgat/regstat/cmd/regstat
C:\> go install -a github.com/vleurgat/regstat/cmd/regstat
````
