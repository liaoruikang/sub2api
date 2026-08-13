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

type AnnouncementGroupPriceChange struct {
	ent.Schema
}

func (AnnouncementGroupPriceChange) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "announcement_group_price_changes"}}
}

func (AnnouncementGroupPriceChange) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("announcement_id"),
		field.Int64("group_id"),
		field.String("group_name").MaxLen(100).Comment("发布时的分组名称快照"),
		field.Float("old_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("new_rate").SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int("sequence").Default(0),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (AnnouncementGroupPriceChange) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("announcement", Announcement.Type).
			Ref("group_price_changes").
			Field("announcement_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (AnnouncementGroupPriceChange) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("announcement_id", "sequence"),
		index.Fields("group_id"),
	}
}
