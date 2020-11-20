# cephfs-xattr-influx

Continuously read CephFS directories extended file attributes (xattr) and store
them in InfluxDB.

## Build

`cephfs-xattr-influx` depends on [Go bindings for
Ceph](https://github.com/ceph/go-ceph) and it was developed and tested to be
used with a Ceph Luminous cluster. Newer Ceph versions may need some adjustments,
contributions a welcome.

```
go build -o cephfs-xattr-influx -tags luminous main.go
```

## Install

Install `cephfs-xattr-influx` as a systemd timer which executes every 15
minutes.

The systemd unit files expects to find the `cephfs-xattr-influx` in `/usr/bin`
it's `config` and `paths.json` files to be prestend in
`/etc/cephfs-xattr-influx`.

### Instructions

Download and install the `cephfs-xattr-influx` binary to `/usr/bin/`:

```
$ curl -L https://github.com/euracresearch/cephfs-xattr-influx/releases/download/<version>/cephfs-xattr-influx-luminous.tar.gz | tar zx && mv cephfs-xattr-influx /usr/bin
```

Copy the systemd unit files to `/etc/systemd/system':

```
$ cp cephfs-xattr-influx.service /etc/systemd/system/
$ cp cephfs-xattr-influx.timer /etc/systemd/system/
```

Enable the unit files:

```
$ systemctl enable /etc/systemd/system/cephfs-xattr-influx.service 
$ systemctl enable /etc/systemd/system/cephfs-xattr-influx.timer 
```

Finally start the timer:

```
$ systemctl start /etc/systemd/system/cephfs-xattr-influx.timer 
```

Show all enabled timers:

```
$ systemctl list-timers
```