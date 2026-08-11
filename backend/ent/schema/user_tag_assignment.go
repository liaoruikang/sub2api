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

// UserTagAssignment holds the edge schema for user tag membership.
type UserTagAssignment struct {
	ent.Schema
}

func (UserTagAssignment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_tag_assignments"},
		field.ID("user_id", "tag_id"),
	}
}

func (UserTagAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("tag_id"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (UserTagAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Unique().
			Required().
			Field("user_id"),
		edge.To("tag", UserTag.Type).
			Unique().
			Required().
			Field("tag_id"),
	}
}

func (UserTagAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tag_id"),
	}
}
