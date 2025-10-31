package repo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"srmt-admin/internal/lib/model/organization-type"
)

const organizationTypeCollection = "organization-types"

type OrganizationTypeRepository struct {
	coll *mongo.Collection
}

func NewOrganizationTypeRepository(db *mongo.Database) *OrganizationTypeRepository {
	return &OrganizationTypeRepository{
		coll: db.Collection(organizationTypeCollection),
	}
}

func (r *OrganizationTypeRepository) GetAllOrganizationTypes(ctx context.Context) ([]organization_type.OrganizationType, error) {
	var organizationTypes []organization_type.OrganizationType

	cur, err := r.coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	if err := cur.All(ctx, &organizationTypes); err != nil {
		return nil, err
	}

	return organizationTypes, nil
}

func (r *OrganizationTypeRepository) SaveOrganizationType(ctx context.Context, name string) (string, error) {
	result, err := r.coll.InsertOne(ctx, organization_type.OrganizationType{Name: name})
	if err != nil {
		return "", err
	}

	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (r *OrganizationTypeRepository) DeleteOrganizationType(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.coll.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *OrganizationTypeRepository) EditOrganizationType(ctx context.Context, id string, name string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.coll.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{"name": name}},
	)
	return err
}
