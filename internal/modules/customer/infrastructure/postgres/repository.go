package postgres

import (
	"context"
	"encoding/json"

	customermodels "github.com/example/medusa-go-clone/internal/modules/customer/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ db *pgxpool.Pool }
type Customer = customermodels.Customer
type Address = customermodels.Address
type AddressInput = customermodels.AddressInput
type AddressUpdate = customermodels.AddressUpdate
type Input = customermodels.Input

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) AccountStatesByEmail(ctx context.Context, email string) ([]bool, error) {
	rows, err := r.db.Query(ctx, `SELECT has_account FROM customer WHERE lower(email)=lower($1) AND deleted_at IS NULL`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := []bool{}
	for rows.Next() {
		var state bool
		if err := rows.Scan(&state); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

func (r *Repository) FindCustomersByEmail(ctx context.Context, email string) ([]customermodels.Customer, error) {
	rows, err := r.db.Query(ctx, `SELECT `+customerColumns+` FROM customer WHERE lower(email)=lower($1) AND deleted_at IS NULL ORDER BY created_at`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []customermodels.Customer{}
	for rows.Next() {
		var c customermodels.Customer
		if err = rows.Scan(&c.ID, &c.CompanyName, &c.FirstName, &c.LastName, &c.Email, &c.Phone, &c.HasAccount, &c.Metadata, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func customerID() string              { return entityid.New("cus") }
func addressID() string               { return entityid.New("cuaddr") }
func customerGroupID() string         { return entityid.New("cusgroup") }
func customerGroupCustomerID() string { return entityid.New("cusgc") }

const customerColumns = `id,company_name,first_name,last_name,email,phone,has_account,metadata,created_by,created_at,updated_at`

func (r *Repository) Create(ctx context.Context, in Input) (Customer, error) {
	if in.Metadata == nil {
		in.Metadata = json.RawMessage(`{}`)
	}
	id := customerID()
	c := Customer{ID: id, Metadata: in.Metadata, Addresses: []Address{}, Groups: []customermodels.CustomerGroup{}}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Customer{}, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO customer(id,company_name,first_name,last_name,email,phone,has_account,metadata,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+customerColumns, id, in.CompanyName, in.FirstName, in.LastName, in.Email, in.Phone, in.HasAccount, in.Metadata, in.CreatedBy).Scan(&c.ID, &c.CompanyName, &c.FirstName, &c.LastName, &c.Email, &c.Phone, &c.HasAccount, &c.Metadata, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return Customer{}, err
	}
	for _, a := range in.Addresses {
		created, err := createAddress(ctx, tx, id, a)
		if err != nil {
			return Customer{}, err
		}
		c.Addresses = append(c.Addresses, created)
	}
	if err := tx.Commit(ctx); err != nil {
		return Customer{}, err
	}
	return c, nil
}
func (r *Repository) createAddress(ctx context.Context, cid string, a AddressInput) (Address, error) {
	return createAddress(ctx, r.db, cid, a)
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func createAddress(ctx context.Context, db queryRower, cid string, a AddressInput) (Address, error) {
	if a.Metadata == nil {
		a.Metadata = json.RawMessage(`{}`)
	}
	id := addressID()
	var out Address
	out.ID = id
	out.CustomerID = cid
	out.AddressInput = a
	err := db.QueryRow(ctx, `INSERT INTO customer_address(id,customer_id,address_name,is_default_shipping,is_default_billing,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id,created_at,updated_at`, id, cid, a.AddressName, a.DefaultShipping, a.DefaultBilling, a.Company, a.FirstName, a.LastName, a.Address1, a.Address2, a.City, a.CountryCode, a.Province, a.PostalCode, a.Phone, a.Metadata).Scan(&out.ID, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}
func (r *Repository) Get(ctx context.Context, id string) (Customer, error) {
	var c Customer
	c.Addresses = []Address{}
	c.Groups = []customermodels.CustomerGroup{}
	err := r.db.QueryRow(ctx, `SELECT `+customerColumns+` FROM customer WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&c.ID, &c.CompanyName, &c.FirstName, &c.LastName, &c.Email, &c.Phone, &c.HasAccount, &c.Metadata, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,customer_id,address_name,is_default_shipping,is_default_billing,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata,created_at,updated_at FROM customer_address WHERE customer_id=$1 AND deleted_at IS NULL ORDER BY created_at`, id)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.CustomerID, &a.AddressName, &a.DefaultShipping, &a.DefaultBilling, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return c, err
		}
		c.Addresses = append(c.Addresses, a)
	}
	if err := rows.Err(); err != nil {
		return c, err
	}
	groupRows, err := r.db.Query(ctx, `SELECT g.id,g.name,g.metadata,g.created_by,g.created_at,g.updated_at FROM customer_group g JOIN customer_group_customer gc ON gc.customer_group_id=g.id WHERE gc.customer_id=$1 AND g.deleted_at IS NULL ORDER BY g.name`, id)
	if err != nil {
		return c, err
	}
	defer groupRows.Close()
	for groupRows.Next() {
		var group customermodels.CustomerGroup
		if err := groupRows.Scan(&group.ID, &group.Name, &group.Metadata, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return c, err
		}
		group.CustomerIDs = []string{id}
		c.Groups = append(c.Groups, group)
	}
	return c, groupRows.Err()
}
func (r *Repository) List(ctx context.Context, limit, offset int) ([]Customer, error) {
	rows, err := r.db.Query(ctx, `SELECT `+customerColumns+` FROM customer WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Customer{}
	for rows.Next() {
		var c Customer
		c.Addresses = []Address{}
		c.Groups = []customermodels.CustomerGroup{}
		if err := rows.Scan(&c.ID, &c.CompanyName, &c.FirstName, &c.LastName, &c.Email, &c.Phone, &c.HasAccount, &c.Metadata, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		list[i], err = r.Get(ctx, list[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM customer WHERE deleted_at IS NULL`).Scan(&count)
	return count, err
}
func (r *Repository) Update(ctx context.Context, id string, in Input) (Customer, error) {
	_, err := r.db.Exec(ctx, `UPDATE customer SET company_name=COALESCE($2,company_name),first_name=COALESCE($3,first_name),last_name=COALESCE($4,last_name),email=COALESCE($5,email),phone=COALESCE($6,phone),metadata=COALESCE($7,metadata),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.CompanyName, in.FirstName, in.LastName, in.Email, in.Phone, in.Metadata)
	if err != nil {
		return Customer{}, err
	}
	return r.Get(ctx, id)
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customer WHERE id=$1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	if _, err := tx.Exec(ctx, `UPDATE customer_address SET deleted_at=now(),updated_at=now() WHERE customer_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM customer_group_customer WHERE customer_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE customer SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RestoreCustomer(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `UPDATE customer SET deleted_at=NULL,updated_at=now() WHERE id=$1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, `UPDATE customer_address SET deleted_at=NULL,updated_at=now() WHERE customer_id=$1 AND deleted_at IS NOT NULL`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) CreateAddress(ctx context.Context, customerID string, in AddressInput) (Address, error) {
	if _, err := r.Get(ctx, customerID); err != nil {
		return Address{}, err
	}
	return r.createAddress(ctx, customerID, in)
}
func (r *Repository) ListAddresses(ctx context.Context, customerID string, limit, offset int) ([]Address, int64, error) {
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM customer_address WHERE customer_id=$1 AND deleted_at IS NULL`, customerID).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,customer_id,address_name,is_default_shipping,is_default_billing,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata,created_at,updated_at FROM customer_address WHERE customer_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`, customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	addresses := []Address{}
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.CustomerID, &a.AddressName, &a.DefaultShipping, &a.DefaultBilling, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		addresses = append(addresses, a)
	}
	return addresses, count, rows.Err()
}
func (r *Repository) GetAddress(ctx context.Context, customerID, addressID string) (Address, error) {
	var a Address
	err := r.db.QueryRow(ctx, `SELECT id,customer_id,address_name,is_default_shipping,is_default_billing,company,first_name,last_name,address_1,address_2,city,country_code,province,postal_code,phone,metadata,created_at,updated_at FROM customer_address WHERE id=$1 AND customer_id=$2 AND deleted_at IS NULL`, addressID, customerID).Scan(&a.ID, &a.CustomerID, &a.AddressName, &a.DefaultShipping, &a.DefaultBilling, &a.Company, &a.FirstName, &a.LastName, &a.Address1, &a.Address2, &a.City, &a.CountryCode, &a.Province, &a.PostalCode, &a.Phone, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}
func (r *Repository) UpdateAddress(ctx context.Context, customerID, addressID string, in AddressUpdate) (Customer, error) {
	tag, err := r.db.Exec(ctx, `UPDATE customer_address SET address_name=COALESCE($3,address_name),is_default_shipping=COALESCE($4,is_default_shipping),is_default_billing=COALESCE($5,is_default_billing),company=COALESCE($6,company),first_name=COALESCE($7,first_name),last_name=COALESCE($8,last_name),address_1=COALESCE($9,address_1),address_2=COALESCE($10,address_2),city=COALESCE($11,city),country_code=COALESCE($12,country_code),province=COALESCE($13,province),postal_code=COALESCE($14,postal_code),phone=COALESCE($15,phone),metadata=COALESCE($16,metadata),updated_at=now() WHERE id=$1 AND customer_id=$2 AND deleted_at IS NULL`, addressID, customerID, in.AddressName, in.DefaultShipping, in.DefaultBilling, in.Company, in.FirstName, in.LastName, in.Address1, in.Address2, in.City, in.CountryCode, in.Province, in.PostalCode, in.Phone, in.Metadata)
	if err != nil {
		return Customer{}, err
	}
	if tag.RowsAffected() == 0 {
		return Customer{}, pgx.ErrNoRows
	}
	return r.Get(ctx, customerID)
}
func (r *Repository) DeleteAddress(ctx context.Context, customerID, addressID string) error {
	tag, err := r.db.Exec(ctx, `UPDATE customer_address SET deleted_at=now(),updated_at=now() WHERE id=$1 AND customer_id=$2 AND deleted_at IS NULL`, addressID, customerID)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) RawDB() *pgxpool.Pool { return r.db }

func (r *Repository) CreateCustomerGroups(ctx context.Context, name string, metadata json.RawMessage, createdBy *string) (customermodels.CustomerGroup, error) {
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}
	group := customermodels.CustomerGroup{ID: customerGroupID(), CustomerIDs: []string{}}
	err := r.db.QueryRow(ctx, `INSERT INTO customer_group(id,name,metadata,created_by) VALUES($1,$2,$3,$4) RETURNING id,name,metadata,created_by,created_at,updated_at`, group.ID, name, metadata, createdBy).Scan(&group.ID, &group.Name, &group.Metadata, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)
	return group, err
}
func (r *Repository) ListCustomerGroups(ctx context.Context) ([]customermodels.CustomerGroup, error) {
	rows, err := r.db.Query(ctx, `SELECT id,name,metadata,created_by,created_at,updated_at FROM customer_group WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []customermodels.CustomerGroup{}
	for rows.Next() {
		var group customermodels.CustomerGroup
		if err := rows.Scan(&group.ID, &group.Name, &group.Metadata, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		group.CustomerIDs = []string{}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range groups {
		customerRows, err := r.db.Query(ctx, `SELECT customer_id FROM customer_group_customer WHERE customer_group_id=$1 ORDER BY created_at`, groups[i].ID)
		if err != nil {
			return nil, err
		}
		for customerRows.Next() {
			var id string
			if err := customerRows.Scan(&id); err != nil {
				customerRows.Close()
				return nil, err
			}
			groups[i].CustomerIDs = append(groups[i].CustomerIDs, id)
		}
		err = customerRows.Err()
		customerRows.Close()
		if err != nil {
			return nil, err
		}
	}
	return groups, nil
}
func (r *Repository) GetCustomerGroup(ctx context.Context, id string) (customermodels.CustomerGroup, error) {
	var group customermodels.CustomerGroup
	group.CustomerIDs = []string{}
	err := r.db.QueryRow(ctx, `SELECT id,name,metadata,created_by,created_at,updated_at FROM customer_group WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&group.ID, &group.Name, &group.Metadata, &group.CreatedBy, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return group, err
	}
	rows, err := r.db.Query(ctx, `SELECT customer_id FROM customer_group_customer WHERE customer_group_id=$1 ORDER BY created_at`, id)
	if err != nil {
		return group, err
	}
	defer rows.Close()
	for rows.Next() {
		var customerID string
		if err := rows.Scan(&customerID); err != nil {
			return group, err
		}
		group.CustomerIDs = append(group.CustomerIDs, customerID)
	}
	return group, rows.Err()
}
func (r *Repository) UpdateCustomerGroup(ctx context.Context, id string, in customermodels.CustomerGroupUpdate) (customermodels.CustomerGroup, error) {
	tag, err := r.db.Exec(ctx, `UPDATE customer_group SET name=COALESCE($2,name),metadata=COALESCE($3,metadata),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id, in.Name, in.Metadata)
	if err != nil {
		return customermodels.CustomerGroup{}, err
	}
	if tag.RowsAffected() == 0 {
		return customermodels.CustomerGroup{}, pgx.ErrNoRows
	}
	return r.GetCustomerGroup(ctx, id)
}
func (r *Repository) DeleteCustomerGroup(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE customer_group SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) RestoreCustomerGroup(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE customer_group SET deleted_at=NULL,updated_at=now() WHERE id=$1 AND deleted_at IS NOT NULL`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *Repository) AddCustomerToGroup(ctx context.Context, customerID, groupID string, createdBy *string) (string, error) {
	id := customerGroupCustomerID()
	_, err := r.db.Exec(ctx, `INSERT INTO customer_group_customer(id,customer_id,customer_group_id,created_by) VALUES($1,$2,$3,$4)`, id, customerID, groupID, createdBy)
	return id, err
}
func (r *Repository) RemoveCustomerFromGroup(ctx context.Context, customerID, groupID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM customer_group_customer WHERE customer_id=$1 AND customer_group_id=$2`, customerID, groupID)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
