package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GrokVideoJob struct {
	ent.Schema
}

func (GrokVideoJob) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "grok_video_jobs"},
	}
}

func (GrokVideoJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("request_id").MaxLen(128).Immutable(),
		field.Int64("user_id"),
		field.Int64("api_key_id").Optional().Nillable(),
		field.Int64("group_id").Optional().Nillable(),
		field.Int64("account_id").Optional().Nillable(),
		field.String("model").MaxLen(128).Default(""),
		field.String("prompt_preview").MaxLen(500).Default(""),
		field.String("status").MaxLen(32).Default("pending"),
		field.Int("progress_percent").Default(0),
		field.String("progress_text").MaxLen(255).Default(""),
		field.String("result_url").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("result_urls").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("cover_image_url").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("last_error_code").Optional().Nillable().MaxLen(128),
		field.String("last_error_message").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("submitted_at").Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("last_polled_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("finished_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GrokVideoJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id").Unique(),
		index.Fields("user_id", "created_at"),
		index.Fields("status", "updated_at"),
		index.Fields("api_key_id", "created_at"),
	}
}
