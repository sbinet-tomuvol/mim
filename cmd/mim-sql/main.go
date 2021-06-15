// Copyright 2020 The go-lpc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/go-lpc/mim/conddb"
	_ "github.com/go-sql-driver/mysql"
)

const (
	dbname = "tmvsrv"
)

func main() {
	log.SetPrefix("mim-sql: ")
	log.SetFlags(0)

	var (
		hrcfg = flag.String("hr-cfg", "", "HardRoc config to inspect")
		dif   = flag.Int("dif", 0x9, "DIF ID to inspect")
	)

	flag.Parse()

	log.Printf("dif: %03d", *dif)
	log.Printf("cfg: %q", *hrcfg)

	db, err := conddb.Open(dbname)
	if err != nil {
		log.Fatalf("could not open MIM db: %+v", err)
	}
	defer db.Close()

	err = doQuery(db, *hrcfg, *dif)
	if err != nil {
		log.Fatalf("could not do query: %+v", err)
	}
}

func doQuery(db *conddb.DB, hrConfig string, difID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if hrConfig == "" {
		v, err := db.LastHRConfig(ctx)
		if err != nil {
			return fmt.Errorf("could not get last hrconfig value: %w", err)
		}
		hrConfig = v
		log.Printf("hrconfig: %q", hrConfig)
	}

	asics, err := db.ASICConfig(ctx, hrConfig, uint8(difID))
	if err != nil {
		return fmt.Errorf("could not get ASIC cfg (hr=%q, id=0x%x): %w",
			hrConfig, uint8(difID), err,
		)
	}
	log.Printf("asics: %d", len(asics))
	//	for i, asic := range asics {
	//		log.Printf("row[%d]: %#v (%q)", i, asic, asic.PreAmpGain)
	//	}

	detid, err := db.LastDetectorID(ctx)
	if err != nil {
		return fmt.Errorf("could not get last det-id: %w", err)
	}
	log.Printf("det-id: %d", detid)
	{
		rows, err := db.QueryContext(ctx, "SELECT dif, asu, iy FROM chambers WHERE detector=? ORDER BY dif", detid)
		if err != nil {
			return fmt.Errorf("could not get chambers definition: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var (
				difid uint32
				asu   uint32
				iy    uint32
			)
			err = rows.Scan(&difid, &asu, &iy)
			if err != nil {
				return fmt.Errorf("could not scan chambers definition: %w", err)
			}
			switch {
			case difid < 100:
				log.Printf(">>> dif=%03d, eda=%02d, slot=%d", difid, asu, iy)
			default:
				log.Printf(">>> dif=%03d, asu=%02d, iy=%d", difid, asu, iy)
			}
		}
	}
	{
		/*
			+------------+----------------------+------+-----+---------+-------+
			| Field      | Type                 | Null | Key | Default | Extra |
			+------------+----------------------+------+-----+---------+-------+
			| detector   | int(11)              | NO   | PRI | NULL    |       |
			| identifier | int(11)              | NO   | PRI | NULL    |       |
			| eda        | int(11)              | NO   | MUL | NULL    |       |
			| iz         | tinyint(1) unsigned  | NO   |     | 0       |       |
			| lv         | smallint(5) unsigned | NO   |     | 0       |       |
			+------------+----------------------+------+-----+---------+-------+
		*/
		rows, err := db.QueryContext(ctx, "SELECT detector, identifier, eda, iz, lv from layers")
		if err != nil {
			return fmt.Errorf("could not get layers definition: %w", err)
		}
		defer rows.Close()

		nrows := 0
		for rows.Next() {
			var (
				det int32
				id  int32
				eda int32
				iz  uint8
				lv  uint32
			)
			err = rows.Scan(&det, &id, &eda, &iz, &lv)
			if err != nil {
				return fmt.Errorf("could not scan layers table: %w", err)
			}
			log.Printf(
				"layer> id=%v, det=%v, eda=%v, iz=%v, lv=%v",
				id, det, eda, iz, lv,
			)
			nrows++
		}
		if nrows == 0 {
			log.Printf("empty layers table")
		}
	}
	{
		/*
			+------------+-------------+------+-----+---------+----------------+
			| Field      | Type        | Null | Key | Default | Extra          |
			+------------+-------------+------+-----+---------+----------------+
			| identifier | int(11)     | NO   | PRI | NULL    | auto_increment |
			| name       | varchar(20) | YES  |     | NULL    |                |
			+------------+-------------+------+-----+---------+----------------+
		*/
		rows, err := db.QueryContext(ctx, "SELECT identifier, name FROM eda")
		if err != nil {
			return fmt.Errorf("could not get EDA definition: %w", err)
		}
		defer rows.Close()

		nrows := 0
		for rows.Next() {
			var (
				id   int32
				name string
			)
			err = rows.Scan(&id, &name)
			if err != nil {
				return fmt.Errorf("could not scan EDA table: %w", err)
			}
			log.Printf("eda> id=%v, name=%v", id, name)
			nrows++
		}
		if nrows == 0 {
			log.Printf("empty eda table")
		}
	}

	{
		rows, err := db.QueryContext(ctx, "SELECT detector, layer, identifier, rfm, chamber, ix, iy, slot, hv, voltage FROM rfm")
		if err != nil {
			return fmt.Errorf("could not get RFM definition: %w", err)
		}
		defer rows.Close()
		/*
			| detector   | int(11)              | NO   | PRI | NULL
			| layer      | int(11)              | NO   | PRI | NULL
			| identifier | int(11)              | NO   | PRI | NULL
			| rfm        | smallint(5) unsigned | NO   |     | 0
			| rtl        | smallint(5) unsigned | NO   |     | 0
			| chamber    | smallint(5) unsigned | NO   |     | 0
			| ix         | tinyint(1) unsigned  | NO   |     | 0
			| iy         | tinyint(1) unsigned  | NO   |     | 0
			| slot       | tinyint(1) unsigned  | NO   |     | 0
			| hv         | smallint(5) unsigned | NO   |     | 0
			| voltage    | float                | NO   |     | 0
		*/

		nrows := 0
		for rows.Next() {
			var (
				det     int32
				layer   int32
				id      int32
				rfm     uint8
				chamber uint8
				ix      uint8
				iy      uint8
				slot    uint8
				hv      uint32
				volt    float64
			)
			err = rows.Scan(&det, &layer, &id, &rfm, &chamber, &ix, &iy, &slot, &hv, &volt)
			if err != nil {
				return fmt.Errorf("could not scan RFM table: %w", err)
			}
			log.Printf(
				"rfm> id=%v, rfm=%v, slot=%v, ix=%v, iy=%v, det=%v, layer=%v, hv=%v, volt=%v",
				id, rfm, slot, ix, iy, det, layer, hv, volt,
			)
			nrows++
		}
		if nrows == 0 {
			log.Printf("empty rfm table")
		}
	}

	daqstates, err := db.DAQStates(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve daqstates: %w", err)
	}
	log.Printf("daqstates: %d", len(daqstates))
	for i, daq := range daqstates {
		log.Printf("row[%d]: %#v", i, daq)
	}

	return nil
}
