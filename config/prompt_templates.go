package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var promptTemplateNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// PromptTemplateRecord 系统提示词模板
type PromptTemplateRecord struct {
	Name        string    `json:"name"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ValidatePromptTemplateName 校验模板名称
func ValidatePromptTemplateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	if !promptTemplateNamePattern.MatchString(name) {
		return fmt.Errorf("模板名称仅允许字母、数字、下划线和连字符")
	}
	return nil
}

func (d *Database) initPromptTemplateTables() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS prompt_templates (
		name TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		description TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	return d.seedPromptTemplatesFromFiles()
}

// seedPromptTemplatesFromFiles 表为空时从 prompts/*.txt 导入种子数据
func (d *Database) seedPromptTemplatesFromFiles() error {
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM prompt_templates`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	promptsDir := "prompts"
	files, err := filepath.Glob(filepath.Join(promptsDir, "*.txt"))
	if err != nil {
		return fmt.Errorf("扫描提示词目录失败: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fileName := filepath.Base(file)
		name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		if err := ValidatePromptTemplateName(name); err != nil {
			continue
		}
		_, _ = d.db.Exec(`
			INSERT OR IGNORE INTO prompt_templates (name, content, description)
			VALUES (?, ?, ?)
		`, name, string(content), "")
	}
	return nil
}

// ListPromptTemplates 分页查询提示词模板
func (d *Database) ListPromptTemplates(page, pageSize int, nameFilter string) ([]PromptTemplateRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	where := ""
	args := []interface{}{}
	if strings.TrimSpace(nameFilter) != "" {
		where = " WHERE name LIKE ?"
		args = append(args, "%"+strings.TrimSpace(nameFilter)+"%")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM prompt_templates" + where
	if err := d.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `SELECT name, content, COALESCE(description, ''), created_at, updated_at
		FROM prompt_templates` + where + ` ORDER BY name ASC LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, offset)

	rows, err := d.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]PromptTemplateRecord, 0)
	for rows.Next() {
		var r PromptTemplateRecord
		if err := rows.Scan(&r.Name, &r.Content, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, r)
	}
	return list, total, nil
}

// GetAllPromptTemplates 获取全部模板（供运行时加载）
func (d *Database) GetAllPromptTemplates() ([]PromptTemplateRecord, error) {
	rows, err := d.db.Query(`SELECT name, content, COALESCE(description, ''), created_at, updated_at
		FROM prompt_templates ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]PromptTemplateRecord, 0)
	for rows.Next() {
		var r PromptTemplateRecord
		if err := rows.Scan(&r.Name, &r.Content, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

// GetPromptTemplate 获取单个模板
func (d *Database) GetPromptTemplate(name string) (*PromptTemplateRecord, error) {
	var r PromptTemplateRecord
	err := d.db.QueryRow(`SELECT name, content, COALESCE(description, ''), created_at, updated_at
		FROM prompt_templates WHERE name = ?`, name).Scan(
		&r.Name, &r.Content, &r.Description, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreatePromptTemplate 创建模板
func (d *Database) CreatePromptTemplate(name, content, description string) error {
	name = strings.TrimSpace(name)
	if err := ValidatePromptTemplateName(name); err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("模板内容不能为空")
	}
	_, err := d.db.Exec(`
		INSERT INTO prompt_templates (name, content, description)
		VALUES (?, ?, ?)
	`, name, content, strings.TrimSpace(description))
	return err
}

// UpdatePromptTemplate 更新模板内容与描述
func (d *Database) UpdatePromptTemplate(name, content, description string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("模板内容不能为空")
	}
	res, err := d.db.Exec(`
		UPDATE prompt_templates
		SET content = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE name = ?
	`, content, strings.TrimSpace(description), name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("模板不存在: %s", name)
	}
	return nil
}

// CountTradersUsingPromptTemplate 统计引用该模板的交易员数量
func (d *Database) CountTradersUsingPromptTemplate(name string) (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM traders WHERE system_prompt_template = ?`, name).Scan(&count)
	return count, err
}

// DeletePromptTemplate 删除模板（有引用时拒绝）
func (d *Database) DeletePromptTemplate(name string) error {
	count, err := d.CountTradersUsingPromptTemplate(name)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("有 %d 个交易员正在使用该模板，无法删除", count)
	}
	res, err := d.db.Exec(`DELETE FROM prompt_templates WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("模板不存在: %s", name)
	}
	return nil
}
