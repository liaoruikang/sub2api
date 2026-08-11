package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GroupUserTag holds the edge schema for tag-derived group access.
type GroupUserTag struct {
	ent.Schema
}

func (GroupUserTag) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "group_user_tags"},
		field.ID("group_id", "tag_id"),
	}
}

func (GroupUserTag) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("group_id"),
		field.Int64("tag_id"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GroupUserTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("group", Group.Type).
			Unique().
			Required().
			Field("group_id"),
		edge.To("tag", UserTag.Type).
			Unique().
			Required().
			Field("tag_id"),
	}
}

func (GroupUserTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tag_id"),
	}
}
