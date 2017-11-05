package dydosyncer

import (
	"context"
	"errors"
	"github.com/digitalocean/godo"
	"time"
)

// DNS Data Syncer
type DydoSyncer struct {
	domain           string
	rtype            string
	rname            string
	record           godo.DomainRecord
	client           *godo.Client
	record_last_sync time.Time
	record_sync_freq time.Duration
	last_rate        godo.Rate
}

// Creates a new DydoSyncer for the domain, `local_record_sync_freq` is how often it will download the remote record
func NewDydoSyncer(domain, rtype, rname string, client *godo.Client, local_record_sync_freq time.Duration) DydoSyncer {
	dydo_syncer := DydoSyncer{
		domain:           domain,
		rtype:            rtype,
		rname:            rname,
		client:           client,
		record_last_sync: time.Time{},
		record_sync_freq: local_record_sync_freq,
	}

	return dydo_syncer
}

func (d *DydoSyncer) fetch() error {
	records, _, err := d.client.Domains.Records(context.TODO(), d.domain, nil)

	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Type == d.rtype && record.Name == d.rname {
			d.record = record
			d.record_last_sync = time.Now()
			return nil
		}
	}

	return errors.New("Unable to find record")
}

func (d *DydoSyncer) update(data string) error {
	editRequest := &godo.DomainRecordEditRequest{
		Data: data,
	}

	_, rate, err := d.client.Domains.EditRecord(context.TODO(), d.domain, d.record.ID, editRequest)

	// TODO(tperson): Type assertion?
	d.last_rate.Limit = rate.Limit
	d.last_rate.Remaining = rate.Remaining
	d.last_rate.Reset = rate.Reset

	return err
}

// Sync the provides data to the record, returns true/false if the sync
// occured and the last data value of the record before sync
func (d *DydoSyncer) Sync(data string) (bool, string, error) {
	lastData := d.record.Data
	if data != d.record.Data {

		if d.record_last_sync.Add(d.record_sync_freq).Before(time.Now()) {
			err := d.fetch()

			if err != nil {
				return false, lastData, nil
			}
		}

		err := d.update(data)

		if err != nil {
			return false, lastData, err
		}

		return true, lastData, nil
	}

	return false, lastData, nil
}
