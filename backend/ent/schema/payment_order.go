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

// PaymentOrder holds the schema definition for the PaymentOrder entity.
type PaymentOrder struct {
	ent.Schema
}

func (PaymentOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment_orders"},
	}
}

func (PaymentOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id").
			Unique().
			Immutable(),
		field.Int64("user_id").
			Immutable(),
		field.String("order_no").
			MaxLen(64).
			Unique().
			Immutable(),
		field.String("trade_no").
			MaxLen(64).
			Optional().
			Nillable(),
		field.Float("amount_usd").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Float("amount_rmb").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}),
		field.Float("exchange_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}),
		field.String("status").
			MaxLen(32).
			Default("pending"),
		field.String("pay_type").
			MaxLen(32).
			Optional().
			Nillable(),
		field.Time("paid_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (PaymentOrder) Edges() []ent.Edge {
	return nil
}

func (PaymentOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("order_no").Unique(),
		index.Fields("status"),
	}
}
