// Copyright 2020 Eurac Research. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// cephfs-xattr-influx will read the given paths JSON file and for each path
// retrieve the extended file attributes of it and store it in InfluxDB.
//
// Sample of paths JSON file:
//  [
//      {
//          "Organisation": "root",
//          "User": "root",
//          "Path": "/"
//      },
//      {
//          "Organisation": "org1",
//          "User": "user2",
//          "Path": "/org1/user2"
//      }
//  ]
//
package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ceph/go-ceph/cephfs"
	"github.com/ceph/go-ceph/rados"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/peterbourgon/ff/v3"
)

func main() {
	fs := flag.NewFlagSet("cephfs-xattr-influx", flag.ExitOnError)
	var (
		influxAddr   = fs.String("influx.addr", "http://localhost:8086", "")
		influxToken  = fs.String("influx.token", "", "InfluxDB authentication token. (For InfluxDB 1.8 use 'username:password')")
		influxOrg    = fs.String("influx.org", "", "InfluxDB Organisation. (For InfluxDB 1.8 leave this empty)")
		influxBucket = fs.String("influx.bucket", "", "InfluxDB Bucket. (For InfluxDB 1.8 use database/retention-policy, skip retention policy if default is used)")
		cephClient   = fs.String("ceph.client", "admin", "Ceph client name.")
		cephKey      = fs.String("ceph.keyring", "", "Ceph client authentication key.")
		cephMons     = fs.String("ceph.mons", "", "Comma seperated list of ceph monitors. (e.g. mon1,mon2)")
		pathsFile    = fs.String("paths", "/etc/cephfs-xattr-influx/paths.json", "JSON file with the paths to monitor.")
		_            = fs.String("config", "", "config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("CEPH_XATTR"),
		ff.WithConfigFileFlag("config"),
		ff.WithConfigFileParser(ff.PlainParser),
	)

	b, err := ioutil.ReadFile(*pathsFile)
	if err != nil {
		log.Fatal(err)
	}

	var paths []*Path
	if err := json.Unmarshal(b, &paths); err != nil {
		log.Fatal(err)
	}

	conn, err := radosConnection(*cephClient)
	if err != nil {
		log.Fatalf("unable to create rados connection: %v", err)
	}

	if err := radosConfiguration(conn, *cephMons); err != nil {
		log.Fatalf("unable to read/set connection configuration: %v", err)
	}

	if *cephKey != "" {
		if err := conn.SetConfigOption("key", *cephKey); err != nil {
			log.Fatalf("unable to set client key: %v", err)
		}
	}
	if err := conn.Connect(); err != nil {
		log.Fatalf("unable to connect: %v\n", err)
	}
	defer conn.Shutdown()

	info, err := cephfs.CreateFromRados(conn)
	if err != nil {
		log.Fatalf("unable to create cephfs mountinfo: %v", err)
	}

	if err := info.Mount(); err != nil {
		log.Fatalf("unable to mount: %v", err)
	}
	defer info.Unmount()

	client := influxdb2.NewClient(*influxAddr, *influxToken)
	writeAPI := client.WriteAPI(*influxOrg, *influxBucket)

	for _, p := range paths {
		attr, err := info.ListXattr(p.Path)
		if err != nil {
			log.Printf("unable to get list of xattr for %q: %v\n", p.Path, err)
			continue
		}

		fields := make(map[string]interface{}, len(attr))
		for _, a := range attr {
			b, err := info.GetXattr(p.Path, a)
			if err != nil {
				continue
			}

			f, err := strconv.ParseFloat(string(b), 64)
			if err != nil {
				log.Printf("unable to convert %q to float: %v", b, err)
			}

			fields[a] = f
		}

		writeAPI.WritePoint(influxdb2.NewPoint(
			"cephfs_xattr",
			p.Tags(),
			fields,
			time.Now(),
		))
	}

	writeAPI.Flush()
	client.Close()
}

func radosConnection(client string) (*rados.Conn, error) {
	if client != "" {
		return rados.NewConnWithUser(client)
	}
	return rados.NewConn()
}

func radosConfiguration(c *rados.Conn, mons string) error {
	if mons != "" {
		return c.SetConfigOption("mon_host", mons)
	}
	return c.ReadDefaultConfigFile()
}

// Path is a CephFS path for which extended file attributes will be optained
// with additional metadata.
type Path struct {
	Organisation string
	User         string
	Path         string
}

func (p *Path) Tags() map[string]string {
	return map[string]string{
		"org":  p.Organisation,
		"user": p.User,
		"path": p.Path,
	}
}
