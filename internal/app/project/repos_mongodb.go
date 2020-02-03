package project

import (
	"context"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gotasma/internal/app/types"
)

type (
	MongoDBRepository struct {
		session *mgo.Session
	}
)

func NewMongoDBRespository(session *mgo.Session) *MongoDBRepository {
	return &MongoDBRepository{
		session: session,
	}
}

func (r *MongoDBRepository) FindByName(ctx context.Context, name string, createrID string) (*types.Project, error) {
	selector := bson.M{"name": name, "creater_id": createrID}
	s := r.session.Clone()
	defer s.Close()
	var project *types.Project
	if err := r.collection(s).Find(selector).One(&project); err != nil {
		return nil, err
	}
	return project, nil
}
func (r *MongoDBRepository) FindByDev(ctx context.Context, devID string) ([]*types.Project, error) {
	selector := bson.M{"dev_id": devID}
	s := r.session.Clone()
	defer s.Close()
	var project []*types.Project
	if err := r.collection(s).Find(selector).All(&project); err != nil {
		return nil, err
	}
	return project, nil
}
func (r *MongoDBRepository) FindByPm(ctx context.Context, pmID string) ([]*types.Project, error) {
	selector := bson.M{"creater_id": pmID}
	s := r.session.Clone()
	defer s.Close()
	var project []*types.Project
	if err := r.collection(s).Find(selector).All(&project); err != nil {
		return nil, err
	}
	return project, nil
}

func (r *MongoDBRepository) Create(ctx context.Context, project *types.Project) (string, error) {
	s := r.session.Clone()
	defer s.Clone()

	project.CreatedAt = time.Now()
	project.UpdateAt = project.CreatedAt

	if err := r.collection(s).Insert(project); err != nil {
		return "", err
	}
	return project.ProjectID, nil
}

func (r *MongoDBRepository) collection(s *mgo.Session) *mgo.Collection {
	return s.DB("").C("project")
}
