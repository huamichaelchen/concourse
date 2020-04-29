package db

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
)

type WorkerBaseResourceType struct {
	Name       string
	WorkerName string
}

type UsedWorkerBaseResourceType struct {
	ID      int
	BaseResourceTypeID int
	Name    string
	Version string

	WorkerName string
}

func (workerBaseResourceType WorkerBaseResourceType) Find(runner sq.Runner) (*UsedWorkerBaseResourceType, bool, error) {
	var id, baseResoureceTypeID int
	var version string
	err := psql.Select("wbrt.id, wbrt.base_resource_type_id, wbrt.version").
		From("worker_base_resource_types wbrt").
		LeftJoin("base_resource_types brt ON brt.id = wbrt.base_resource_type_id").
		LeftJoin("workers w ON w.name = wbrt.worker_name").
		Where(sq.Eq{
			"brt.name":         workerBaseResourceType.Name,
			"wbrt.worker_name": workerBaseResourceType.WorkerName,
		}).
		RunWith(runner).
		QueryRow().
		Scan(&id,
			&baseResoureceTypeID,
			&version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &UsedWorkerBaseResourceType{
		ID:         id,
		BaseResourceTypeID: baseResoureceTypeID,
		Name:       workerBaseResourceType.Name,
		Version:    version,
		WorkerName: workerBaseResourceType.WorkerName,
	}, true, nil
}
