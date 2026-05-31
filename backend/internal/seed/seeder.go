package seed

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type categorySeed struct {
	Name string
	Slug string
}

type productSeed struct {
	Name        string
	Description string
	Price       int64
	Stock       int
	CategoryID  string
	Style       string
	Color       string
	ImageURL    string
}

func Run(ctx context.Context, db *pgxpool.Pool) error {
	categories := []categorySeed{
		{Name: "Tops", Slug: "tops"},
		{Name: "Bottoms", Slug: "bottoms"},
		{Name: "Outerwear", Slug: "outerwear"},
		{Name: "Footwear", Slug: "footwear"},
		{Name: "Accessories", Slug: "accessories"},
		{Name: "Dresses", Slug: "dresses"},
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	categoryIDs := make(map[string]string, len(categories))
	for _, c := range categories {
		id := deterministicID("category:" + c.Slug)
		categoryIDs[c.Slug] = id

		_, err := tx.Exec(ctx, `
			INSERT INTO categories (id, name, slug)
			VALUES ($1, $2, $3)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name,
			    slug = EXCLUDED.slug,
			    updated_at = NOW()
		`, id, c.Name, c.Slug)
		if err != nil {
			return fmt.Errorf("upsert category %s: %w", c.Slug, err)
		}
	}

	products := buildProducts(categoryIDs)
	for _, p := range products {
		id := deterministicID("product:" + slugify(p.Name))
		_, err := tx.Exec(ctx, `
			INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name,
			    description = EXCLUDED.description,
			    price = EXCLUDED.price,
			    stock = EXCLUDED.stock,
			    category_id = EXCLUDED.category_id,
			    style = EXCLUDED.style,
			    color = EXCLUDED.color,
			    image_url = EXCLUDED.image_url,
			    updated_at = NOW()
		`, id, p.Name, p.Description, p.Price, p.Stock, p.CategoryID, p.Style, p.Color, p.ImageURL)
		if err != nil {
			return fmt.Errorf("upsert product %s: %w", p.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func buildProducts(categoryIDs map[string]string) []productSeed {
	styles := []string{"streetwear", "minimal", "korean", "formal", "casual", "sporty"}
	colors := []string{"black", "white", "beige", "blue", "gray", "brown"}

	type catalogTemplate struct {
		Label        string
		CategorySlug string
		BasePrice    int64
	}

	templates := []catalogTemplate{
		{Label: "Oversized Tee", CategorySlug: "tops", BasePrice: 290000},
		{Label: "Wide Pants", CategorySlug: "bottoms", BasePrice: 540000},
		{Label: "Layered Jacket", CategorySlug: "outerwear", BasePrice: 850000},
		{Label: "Urban Sneakers", CategorySlug: "footwear", BasePrice: 990000},
		{Label: "Structured Bag", CategorySlug: "accessories", BasePrice: 640000},
		{Label: "Flow Dress", CategorySlug: "dresses", BasePrice: 760000},
	}

	products := make([]productSeed, 0, len(styles)*len(colors))
	for si, style := range styles {
		for ci, color := range colors {
			template := templates[(si+ci)%len(templates)]
			price := template.BasePrice + int64(si*120000+ci*70000)
			stock := 4 + ((si + 3) * (ci + 5) % 27)

			name := fmt.Sprintf("%s %s %s", title(style), title(color), template.Label)
			description := fmt.Sprintf(
				"%s look with %s tone. Designed for daily wear with balanced comfort and silhouette.",
				title(style),
				color,
			)

			products = append(products, productSeed{
				Name:        name,
				Description: description,
				Price:       price,
				Stock:       stock,
				CategoryID:  categoryIDs[template.CategorySlug],
				Style:       style,
				Color:       color,
				ImageURL:    fmt.Sprintf("https://picsum.photos/seed/stylemind-%s-%s/640/800", style, color),
			})
		}
	}

	return products
}

func deterministicID(input string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("stylemind:"+input)).String()
}

func slugify(input string) string {
	clean := strings.ToLower(strings.TrimSpace(input))
	clean = strings.ReplaceAll(clean, " ", "-")
	return clean
}

func title(input string) string {
	if input == "" {
		return input
	}
	return strings.ToUpper(input[:1]) + input[1:]
}
