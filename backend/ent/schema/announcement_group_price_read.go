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

type AnnouncementGroupPriceRead struct {
	ent.Schema
}

func (AnnouncementGroupPriceRead) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "announcement_group_price_reads"}}
}

func (AnnouncementGroupPriceRead) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("announcement_id"),
		field.Int64("user_id"),
		field.Int64("group_id"),
		field.Time("read_at").Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (AnnouncementGroupPriceRead) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("announcement", Announcement.Type).
			Ref("group_price_reads").
			Field("announcement_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("user", User.Type).
			Ref("announcement_group_price_reads").
			Field("user_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (AnnouncementGroupPriceRead) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "announcement_id"),
		index.Fields("group_id"),
		index.Fields("announcement_id", "user_id", "group_id").Unique(),
	}
}
