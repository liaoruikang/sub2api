package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserTag holds the schema definition for reusable user authorization tags.
type UserTag struct {
	ent.Schema
}

func (UserTag) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_tags"},
	}
}

func (UserTag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (UserTag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty(),
	}
}

func (UserTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("users", User.Type).
			Ref("tags").
			Through("user_tag_assignments", UserTagAssignment.Type),
		edge.From("groups", Group.Type).
			Ref("user_tags").
			Through("group_user_tags", GroupUserTag.Type),
	}
}

func (UserTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("deleted_at"),
	}
}
