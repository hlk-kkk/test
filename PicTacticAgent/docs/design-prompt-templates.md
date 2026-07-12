# Prompt Templates Feature - Design Document

## 1. Overview

This document defines the complete design for the **Prompt Templates** feature in PicTacticAgent. Users can save, manage, and reuse prompt templates to quickly generate images with pre-configured prompts and settings.

**Scope (v1):** Private templates only (`is_public` always `False`). Public/shared templates are deferred to v2.

---

## 2. Database

### 2.1 Table: `prompt_templates`

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `String(36)` | PK | UUID v4 |
| `user_id` | `String(36)` | FK -> `users.user_id` ON DELETE CASCADE, indexed | Owner |
| `title` | `String(200)` | NOT NULL | Template title |
| `description` | `Text` | nullable | Optional description |
| `prompt_text` | `Text` | NOT NULL | The prompt content |
| `category` | `String(50)` | NOT NULL, indexed, default `"other"` | One of: `cover`, `portrait`, `landscape`, `abstract`, `product`, `other` |
| `tags` | `JSON` | default `[]` | `list[str]` of user-defined tags |
| `preview_image_url` | `String(512)` | nullable | URL of a preview image |
| `config` | `JSON` | nullable | Default generation config override (see 2.2) |
| `usage_count` | `Integer` | default `0` | Times used to generate |
| `is_public` | `Boolean` | default `False` | v1: always `False` |
| `created_at` | `DateTime` | NOT NULL, default `utcnow` | Creation timestamp |
| `updated_at` | `DateTime` | NOT NULL, default `utcnow`, onupdate `utcnow` | Last update timestamp |

### 2.2 `config` JSON Schema

When non-null, the `config` field stores default generation parameters that override global defaults when generating from this template. All fields are optional:

```json
{
  "max_rounds": 3,
  "images_per_round": 5,
  "quality_threshold": 0.7,
  "aspect_ratio": "16:9",
  "size": "2K",
  "enhance_prompt": true,
  "enable_ai_scoring": true,
  "provider": "gemini"
}
```

### 2.3 ORM Model

File: `backend/src/pictactic/db/models.py`

```python
class PromptTemplate(Base):
    """Prompt template model."""

    __tablename__ = "prompt_templates"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    user_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="CASCADE"), index=True
    )
    title: Mapped[str] = mapped_column(String(200))
    description: Mapped[str | None] = mapped_column(Text, nullable=True)
    prompt_text: Mapped[str] = mapped_column(Text)
    category: Mapped[str] = mapped_column(String(50), default="other", index=True)
    tags: Mapped[list | None] = mapped_column(JSON, default=list)
    preview_image_url: Mapped[str | None] = mapped_column(String(512), nullable=True)
    config: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    usage_count: Mapped[int] = mapped_column(Integer, default=0)
    is_public: Mapped[bool] = mapped_column(Boolean, default=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=datetime.utcnow, onupdate=datetime.utcnow
    )

    user: Mapped["User"] = relationship(back_populates="templates")
```

Add to `User` model:

```python
templates: Mapped[list["PromptTemplate"]] = relationship(
    back_populates="user", cascade="all, delete-orphan"
)
```

### 2.4 Category Enum

File: `backend/src/pictactic/models/enums.py`

```python
class TemplateCategory(str, Enum):
    """Prompt template categories."""

    COVER = "cover"
    PORTRAIT = "portrait"
    LANDSCAPE = "landscape"
    ABSTRACT = "abstract"
    PRODUCT = "product"
    OTHER = "other"
```

### 2.5 Migration

File: `backend/src/pictactic/db/engine.py` -- add to `_apply_migrations()`:

The `prompt_templates` table is a new table, so `Base.metadata.create_all` (already called in `init_db`) will create it automatically when the model is registered. **No ALTER TABLE migration is needed** -- the table will be created on first startup after the model is added.

However, if the DB already exists without this table, `create_all` handles it correctly since SQLAlchemy's `create_all` uses `CREATE TABLE IF NOT EXISTS` semantics.

---

## 3. Pydantic Schemas

File: `backend/src/pictactic/models/schemas.py`

### 3.1 TemplateCreate

```python
class TemplateConfigOverride(BaseModel):
    """Optional generation config to store with a template."""

    max_rounds: Optional[int] = Field(default=None, ge=1, le=10)
    images_per_round: Optional[int] = Field(default=None, ge=1, le=10)
    quality_threshold: Optional[float] = Field(default=None, ge=0.0, le=1.0)
    aspect_ratio: Optional[str] = None
    size: Optional[str] = None
    enhance_prompt: Optional[bool] = None
    enable_ai_scoring: Optional[bool] = None
    provider: Optional[str] = None


class TemplateCreate(BaseModel):
    """Request body for creating a prompt template."""

    title: str = Field(..., min_length=1, max_length=200)
    description: Optional[str] = Field(default=None, max_length=2000)
    prompt_text: str = Field(..., min_length=1)
    category: TemplateCategory = Field(default=TemplateCategory.OTHER)
    tags: list[str] = Field(default_factory=list, max_length=20)
    preview_image_url: Optional[str] = Field(default=None, max_length=512)
    config: Optional[TemplateConfigOverride] = None
```

### 3.2 TemplateUpdate

```python
class TemplateUpdate(BaseModel):
    """Request body for updating a prompt template. All fields optional."""

    title: Optional[str] = Field(default=None, min_length=1, max_length=200)
    description: Optional[str] = Field(default=None, max_length=2000)
    prompt_text: Optional[str] = Field(default=None, min_length=1)
    category: Optional[TemplateCategory] = None
    tags: Optional[list[str]] = Field(default=None, max_length=20)
    preview_image_url: Optional[str] = Field(default=None, max_length=512)
    config: Optional[TemplateConfigOverride] = None
```

### 3.3 TemplateResponse

```python
class TemplateResponse(BaseModel):
    """Single template in API responses."""

    id: str
    user_id: str
    title: str
    description: Optional[str] = None
    prompt_text: str
    category: str
    tags: list[str] = Field(default_factory=list)
    preview_image_url: Optional[str] = None
    config: Optional[dict] = None
    usage_count: int = 0
    is_public: bool = False
    created_at: str  # ISO 8601
    updated_at: str  # ISO 8601
```

### 3.4 TemplateListResponse

```python
class TemplateListResponse(BaseModel):
    """Paginated list of templates."""

    items: list[TemplateResponse]
    total: int
    page: int
    page_size: int
```

### 3.5 TemplateGenerateRequest

```python
class TemplateGenerateRequest(BaseModel):
    """Request to generate images from a template, with optional overrides."""

    config_overrides: Optional[TemplateConfigOverride] = None
    template_images: list[str] = Field(
        default_factory=list, description="Optional template images for I2I mode"
    )
```

---

## 4. Repository

File: `backend/src/pictactic/db/template_repository.py`

```python
class TemplateRepository:
    """CRUD operations for prompt templates."""

    async def create(self, user_id: str, data: TemplateCreate) -> PromptTemplate:
        """Create a new template. Returns the created ORM object."""
        ...

    async def get_by_id(self, template_id: str, user_id: str) -> PromptTemplate | None:
        """Get a single template by ID, scoped to user."""
        ...

    async def list_templates(
        self,
        user_id: str,
        search: str | None = None,
        category: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[PromptTemplate], int]:
        """
        List templates with optional search/filter.
        Returns (items, total_count).
        Search matches against title, description, prompt_text, and tags.
        """
        ...

    async def update(
        self, template_id: str, user_id: str, data: TemplateUpdate
    ) -> PromptTemplate | None:
        """Update a template. Returns updated object or None if not found."""
        ...

    async def delete(self, template_id: str, user_id: str) -> bool:
        """Delete a template. Returns True if deleted, False if not found."""
        ...

    async def increment_usage(self, template_id: str) -> None:
        """Increment the usage_count by 1."""
        ...
```

### 4.1 Search Implementation

The `list_templates` method applies search across multiple fields using SQLAlchemy `or_` + `ilike`:

```python
if search:
    pattern = f"%{search}%"
    stmt = stmt.where(
        or_(
            PromptTemplate.title.ilike(pattern),
            PromptTemplate.description.ilike(pattern),
            PromptTemplate.prompt_text.ilike(pattern),
        )
    )
```

Tag search uses JSON containment (for SQLite, cast tags to string and use `LIKE`):

```python
# SQLite-compatible tag search
if search:
    from sqlalchemy import cast, String as SAString
    stmt = stmt.where(
        or_(
            PromptTemplate.title.ilike(pattern),
            PromptTemplate.description.ilike(pattern),
            PromptTemplate.prompt_text.ilike(pattern),
            cast(PromptTemplate.tags, SAString).ilike(pattern),
        )
    )
```

### 4.2 Pagination

```python
total_stmt = select(func.count()).select_from(stmt.subquery())
total = (await session.execute(total_stmt)).scalar()

stmt = stmt.order_by(desc(PromptTemplate.updated_at))
stmt = stmt.offset((page - 1) * page_size).limit(page_size)
```

---

## 5. Service Layer

File: `backend/src/pictactic/services/template_service.py`

```python
class TemplateService:
    def __init__(self):
        self.repo = TemplateRepository()

    async def create_template(self, user_id: str, data: TemplateCreate) -> TemplateResponse:
        """Create template and return response schema."""
        ...

    async def get_template(self, template_id: str, user_id: str) -> TemplateResponse:
        """Get template or raise TemplateNotFoundError."""
        ...

    async def list_templates(
        self, user_id: str, search: str | None, category: str | None,
        page: int, page_size: int,
    ) -> TemplateListResponse:
        """List with pagination."""
        ...

    async def update_template(
        self, template_id: str, user_id: str, data: TemplateUpdate,
    ) -> TemplateResponse:
        """Update or raise TemplateNotFoundError."""
        ...

    async def delete_template(self, template_id: str, user_id: str) -> None:
        """Delete or raise TemplateNotFoundError."""
        ...

    async def generate_from_template(
        self, template_id: str, user_id: str, request: TemplateGenerateRequest,
        background_tasks: BackgroundTasks,
    ) -> TaskCreate:
        """
        Load template, merge config with overrides, create generation task.
        Increments usage_count.
        """
        ...
```

### 5.1 Config Merge Logic (generate_from_template)

```python
# 1. Start with global defaults
base_config = {
    "max_rounds": settings.default_max_rounds,
    "images_per_round": settings.default_images_per_round,
    "quality_threshold": settings.default_quality_threshold,
    "aspect_ratio": settings.default_aspect_ratio,
    "size": settings.default_image_size,
    "enhance_prompt": True,
    "enable_ai_scoring": True,
    "provider": settings.default_provider,
}

# 2. Apply template.config (if set)
if template.config:
    for k, v in template.config.items():
        if v is not None:
            base_config[k] = v

# 3. Apply request overrides (if set)
if request.config_overrides:
    overrides = request.config_overrides.model_dump(exclude_none=True)
    base_config.update(overrides)

# 4. Build GenerationRequest
gen_request = GenerationRequest(
    prompt=template.prompt_text,
    template_images=request.template_images,
    **base_config,
)
```

### 5.2 Exception

Add to `backend/src/pictactic/core/exceptions.py`:

```python
class TemplateNotFoundError(PicTacticError):
    """Template not found."""

    def __init__(self, template_id: str):
        super().__init__(f"Template not found: {template_id}")
        self.template_id = template_id
```

### 5.3 Singleton Access

```python
_template_service: TemplateService | None = None

def get_template_service() -> TemplateService:
    global _template_service
    if _template_service is None:
        _template_service = TemplateService()
    return _template_service
```

---

## 6. API Routes

File: `backend/src/pictactic/api/routes/templates.py`

Prefix: `/api/v1/templates`

All endpoints require authentication (`Depends(get_current_user)`).

### 6.1 POST `/` -- Create Template

**Request:**
```json
{
  "title": "Tech Blog Cover",
  "description": "Modern tech blog post cover image",
  "prompt_text": "A modern minimalist tech blog cover with gradient background...",
  "category": "cover",
  "tags": ["tech", "blog", "minimalist"],
  "preview_image_url": null,
  "config": {
    "aspect_ratio": "16:9",
    "size": "2K",
    "images_per_round": 3
  }
}
```

**Response (201):**
```json
{
  "id": "uuid-here",
  "user_id": "user-uuid",
  "title": "Tech Blog Cover",
  "description": "Modern tech blog post cover image",
  "prompt_text": "A modern minimalist tech blog cover with gradient background...",
  "category": "cover",
  "tags": ["tech", "blog", "minimalist"],
  "preview_image_url": null,
  "config": { "aspect_ratio": "16:9", "size": "2K", "images_per_round": 3 },
  "usage_count": 0,
  "is_public": false,
  "created_at": "2025-01-01T00:00:00",
  "updated_at": "2025-01-01T00:00:00"
}
```

**Errors:**
| Code | Condition |
|---|---|
| 400 | Validation error (empty title, invalid category, etc.) |
| 401 | Not authenticated |

### 6.2 GET `/` -- List Templates

**Query Parameters:**
| Param | Type | Default | Description |
|---|---|---|---|
| `search` | `str` | `None` | Search in title, description, prompt_text, tags |
| `category` | `str` | `None` | Filter by category enum value |
| `page` | `int` | `1` | Page number (1-based) |
| `page_size` | `int` | `20` | Items per page (max 100) |

**Response (200):**
```json
{
  "items": [ /* TemplateResponse objects */ ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

**Errors:**
| Code | Condition |
|---|---|
| 401 | Not authenticated |

### 6.3 GET `/{template_id}` -- Get Template

**Response (200):** `TemplateResponse`

**Errors:**
| Code | Condition |
|---|---|
| 401 | Not authenticated |
| 404 | Template not found or not owned by user |

### 6.4 PUT `/{template_id}` -- Update Template

**Request:** `TemplateUpdate` (partial update, only provided fields are changed)

**Response (200):** `TemplateResponse`

**Errors:**
| Code | Condition |
|---|---|
| 400 | Validation error |
| 401 | Not authenticated |
| 404 | Template not found or not owned by user |

### 6.5 DELETE `/{template_id}` -- Delete Template

**Response (200):**
```json
{ "success": true }
```

**Errors:**
| Code | Condition |
|---|---|
| 401 | Not authenticated |
| 404 | Template not found or not owned by user |

### 6.6 POST `/{template_id}/generate` -- Generate from Template

**Request:**
```json
{
  "config_overrides": {
    "max_rounds": 5,
    "images_per_round": 3
  },
  "template_images": []
}
```

**Response (201):**
```json
{
  "task_id": "task-uuid",
  "status": "pending",
  "message": "Task created"
}
```

**Errors:**
| Code | Condition |
|---|---|
| 400 | Invalid config overrides |
| 401 | Not authenticated |
| 404 | Template not found or not owned by user |

### 6.7 Route Registration

In `backend/src/pictactic/api/app.py`, add:

```python
from .routes import auth, generation, health, templates

# In create_app():
application.include_router(
    templates.router,
    prefix="/api/v1/templates",
    tags=["Templates"],
)
```

### 6.8 Router Implementation Skeleton

```python
router = APIRouter()


@router.post("/", response_model=TemplateResponse, status_code=201)
async def create_template(
    data: TemplateCreate,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    return await service.create_template(user.user_id, data)


@router.get("/", response_model=TemplateListResponse)
async def list_templates(
    search: str | None = None,
    category: str | None = None,
    page: int = Query(default=1, ge=1),
    page_size: int = Query(default=20, ge=1, le=100),
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    return await service.list_templates(
        user_id=user.user_id,
        search=search,
        category=category,
        page=page,
        page_size=page_size,
    )


@router.get("/{template_id}", response_model=TemplateResponse)
async def get_template(
    template_id: str,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        return await service.get_template(template_id, user.user_id)
    except TemplateNotFoundError:
        raise HTTPException(status_code=404, detail="Template not found")


@router.put("/{template_id}", response_model=TemplateResponse)
async def update_template(
    template_id: str,
    data: TemplateUpdate,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        return await service.update_template(template_id, user.user_id, data)
    except TemplateNotFoundError:
        raise HTTPException(status_code=404, detail="Template not found")


@router.delete("/{template_id}")
async def delete_template(
    template_id: str,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        await service.delete_template(template_id, user.user_id)
        return {"success": True}
    except TemplateNotFoundError:
        raise HTTPException(status_code=404, detail="Template not found")


@router.post("/{template_id}/generate", response_model=TaskCreate, status_code=201)
async def generate_from_template(
    template_id: str,
    request: TemplateGenerateRequest,
    background_tasks: BackgroundTasks,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        return await service.generate_from_template(
            template_id, user.user_id, request, background_tasks,
        )
    except TemplateNotFoundError:
        raise HTTPException(status_code=404, detail="Template not found")
```

---

## 7. Frontend

### 7.1 API Client Methods

File: `frontend/src/lib/api.js` -- add to `ApiClient` class:

```javascript
// ========== Templates ==========

async createTemplate(data) {
  const response = await this._fetchWithAuth(`${this.baseUrl}/api/v1/templates/`, {
    method: "POST",
    headers: this._headers({ "Content-Type": "application/json" }),
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to create template");
  }
  return response.json();
}

async listTemplates({ search, category, page = 1, pageSize = 20 } = {}) {
  const params = new URLSearchParams();
  if (search) params.set("search", search);
  if (category) params.set("category", category);
  params.set("page", String(page));
  params.set("page_size", String(pageSize));

  const response = await this._fetchWithAuth(
    `${this.baseUrl}/api/v1/templates/?${params}`,
    { headers: this._headers() }
  );
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to list templates");
  }
  return response.json();
}

async getTemplate(templateId) {
  const response = await this._fetchWithAuth(
    `${this.baseUrl}/api/v1/templates/${templateId}`,
    { headers: this._headers() }
  );
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to get template");
  }
  return response.json();
}

async updateTemplate(templateId, data) {
  const response = await this._fetchWithAuth(
    `${this.baseUrl}/api/v1/templates/${templateId}`,
    {
      method: "PUT",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify(data),
    }
  );
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to update template");
  }
  return response.json();
}

async deleteTemplate(templateId) {
  const response = await this._fetchWithAuth(
    `${this.baseUrl}/api/v1/templates/${templateId}`,
    { method: "DELETE", headers: this._headers() }
  );
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to delete template");
  }
  return response.json();
}

async generateFromTemplate(templateId, data = {}) {
  const response = await this._fetchWithAuth(
    `${this.baseUrl}/api/v1/templates/${templateId}/generate`,
    {
      method: "POST",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify(data),
    }
  );
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to generate from template");
  }
  return response.json();
}
```

### 7.2 `useTemplates` Hook

File: `frontend/src/hooks/useTemplates.js`

```javascript
import { useState, useCallback } from "react";
import { apiClient } from "@/lib/api";

const CATEGORIES = ["cover", "portrait", "landscape", "abstract", "product", "other"];

export { CATEGORIES };

export function useTemplates() {
  const [templates, setTemplates] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Filters
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState(null);

  const fetchTemplates = useCallback(async (params = {}) => {
    setLoading(true);
    setError(null);
    try {
      const result = await apiClient.listTemplates({
        search: params.search ?? search,
        category: params.category ?? category,
        page: params.page ?? page,
        pageSize,
      });
      setTemplates(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [search, category, page, pageSize]);

  const createTemplate = useCallback(async (data) => {
    const created = await apiClient.createTemplate(data);
    await fetchTemplates();
    return created;
  }, [fetchTemplates]);

  const updateTemplate = useCallback(async (templateId, data) => {
    const updated = await apiClient.updateTemplate(templateId, data);
    await fetchTemplates();
    return updated;
  }, [fetchTemplates]);

  const deleteTemplate = useCallback(async (templateId) => {
    await apiClient.deleteTemplate(templateId);
    await fetchTemplates();
  }, [fetchTemplates]);

  const generateFromTemplate = useCallback(async (templateId, overrides) => {
    return await apiClient.generateFromTemplate(templateId, overrides);
  }, []);

  const handleSearch = useCallback((value) => {
    setSearch(value);
    setPage(1);
    fetchTemplates({ search: value, page: 1 });
  }, [fetchTemplates]);

  const handleCategoryChange = useCallback((cat) => {
    setCategory(cat);
    setPage(1);
    fetchTemplates({ category: cat, page: 1 });
  }, [fetchTemplates]);

  const handlePageChange = useCallback((newPage) => {
    setPage(newPage);
    fetchTemplates({ page: newPage });
  }, [fetchTemplates]);

  return {
    templates,
    total,
    page,
    pageSize,
    loading,
    error,
    search,
    category,
    fetchTemplates,
    createTemplate,
    updateTemplate,
    deleteTemplate,
    generateFromTemplate,
    handleSearch,
    handleCategoryChange,
    handlePageChange,
    CATEGORIES,
  };
}
```

### 7.3 Page & Component Structure

```
frontend/src/
  pages/
    TemplatesPage.jsx          # Full-page template library
  components/
    templates/
      TemplateList.jsx         # Grid of TemplateCard items
      TemplateCard.jsx         # Single template card
      TemplateDialog.jsx       # Create/Edit dialog (modal)
      SaveAsTemplateButton.jsx # Button shown in image results
      TemplateFilters.jsx      # Search bar + category filter tabs
```

### 7.4 Routing

File: `frontend/src/main.jsx` -- add route inside `AppRoutes`:

```jsx
import TemplatesPage from '@/pages/TemplatesPage'

// Inside <Routes>:
<Route path="/app/templates" element={<ProtectedRoute><TemplatesPage /></ProtectedRoute>} />
```

### 7.5 Navigation

File: `frontend/src/components/layout/Header.jsx` -- add a navigation link:

```jsx
import { FileText } from "lucide-react";
import { useNavigate, useLocation } from "react-router-dom";

// Inside the header actions area, before the user menu:
const navigate = useNavigate();
const location = useLocation();

<button
  onClick={() => navigate("/app/templates")}
  className={`flex items-center gap-1.5 rounded-lg border border-border/50 px-3 py-1.5 text-xs font-medium transition-all duration-200 hover:bg-secondary hover:border-border ${
    location.pathname === "/app/templates"
      ? "bg-secondary text-foreground border-border"
      : "bg-card text-foreground"
  }`}
>
  <FileText className="h-3.5 w-3.5" />
  {t("nav.templates")}
</button>
```

### 7.6 Component Specifications

#### TemplatesPage

The top-level page component. Layout:

```
+------------------------------------------------------------------+
|  Header (with navigation back to /app)                           |
+------------------------------------------------------------------+
|  TemplateFilters (search bar + category tabs)                    |
|  [+ New Template] button                                         |
+------------------------------------------------------------------+
|  TemplateList                                                    |
|    TemplateCard  TemplateCard  TemplateCard                      |
|    TemplateCard  TemplateCard  TemplateCard                      |
|    ...                                                           |
+------------------------------------------------------------------+
|  Pagination                                                      |
+------------------------------------------------------------------+
```

- Uses `useTemplates()` hook
- Calls `fetchTemplates()` on mount
- Shows empty state when no templates exist
- Opens `TemplateDialog` for create/edit

#### TemplateCard

Displays a single template as a card:

```
+-----------------------------------+
|  [Preview Image / Placeholder]    |
|  ---                              |
|  Title                            |
|  Category badge   Usage count     |
|  Tags (chips)                     |
|  ---                              |
|  [Generate] [Edit] [Delete]       |
+-----------------------------------+
```

Props:
- `template` -- TemplateResponse object
- `onGenerate(template)` -- callback
- `onEdit(template)` -- callback
- `onDelete(template)` -- callback

#### TemplateDialog

Modal dialog for creating or editing a template. Uses controlled form state.

Props:
- `open` -- boolean
- `onClose()` -- callback
- `template` -- TemplateResponse | null (null = create mode, object = edit mode)
- `onSave(data)` -- callback

Form fields:
1. Title (text input, required)
2. Description (textarea, optional)
3. Prompt text (textarea, required)
4. Category (select dropdown)
5. Tags (tag input -- type and press Enter to add)
6. Config overrides (collapsible section with optional generation settings)

#### SaveAsTemplateButton

A button component shown on the results page alongside generated images. When clicked, opens `TemplateDialog` pre-filled with:
- `prompt_text` = the prompt used for generation
- `config` = the generation config used

Props:
- `prompt` -- string (the prompt used)
- `config` -- object (generation config)
- `previewImageUrl` -- string | null (best image URL)

#### TemplateFilters

Search and category filter UI:

```
+------------------------------------------------------------------+
|  [Search input _______________]                                  |
|  [All] [Cover] [Portrait] [Landscape] [Abstract] [Product] [Other]
+------------------------------------------------------------------+
```

Props:
- `search` -- string
- `category` -- string | null
- `onSearchChange(value)` -- callback
- `onCategoryChange(category)` -- callback

Uses debounced search (300ms).

### 7.7 i18n Keys

Add to both `en` and `zh` locale files:

```json
{
  "nav.templates": "Templates" / "模板库",
  "templates.title": "Prompt Templates" / "提示词模板",
  "templates.create": "New Template" / "新建模板",
  "templates.edit": "Edit Template" / "编辑模板",
  "templates.delete": "Delete Template" / "删除模板",
  "templates.deleteConfirm": "Are you sure you want to delete this template?" / "确定要删除此模板吗?",
  "templates.generate": "Generate" / "生成",
  "templates.saveAs": "Save as Template" / "保存为模板",
  "templates.empty": "No templates yet. Create your first template!" / "暂无模板, 创建你的第一个模板吧!",
  "templates.searchPlaceholder": "Search templates..." / "搜索模板...",
  "templates.form.title": "Title" / "标题",
  "templates.form.description": "Description" / "描述",
  "templates.form.promptText": "Prompt" / "提示词",
  "templates.form.category": "Category" / "分类",
  "templates.form.tags": "Tags" / "标签",
  "templates.form.config": "Generation Config" / "生成配置",
  "templates.category.all": "All" / "全部",
  "templates.category.cover": "Cover" / "封面",
  "templates.category.portrait": "Portrait" / "人像",
  "templates.category.landscape": "Landscape" / "风景",
  "templates.category.abstract": "Abstract" / "抽象",
  "templates.category.product": "Product" / "产品",
  "templates.category.other": "Other" / "其他",
  "templates.usageCount": "Used {count} times" / "已使用 {count} 次"
}
```

---

## 8. File Change Summary

### Backend -- New Files

| File | Description |
|---|---|
| `backend/src/pictactic/db/template_repository.py` | TemplateRepository class |
| `backend/src/pictactic/services/template_service.py` | TemplateService class |
| `backend/src/pictactic/api/routes/templates.py` | API route handlers |

### Backend -- Modified Files

| File | Changes |
|---|---|
| `backend/src/pictactic/db/models.py` | Add `PromptTemplate` model; add `templates` relationship to `User` |
| `backend/src/pictactic/models/enums.py` | Add `TemplateCategory` enum |
| `backend/src/pictactic/models/schemas.py` | Add `TemplateConfigOverride`, `TemplateCreate`, `TemplateUpdate`, `TemplateResponse`, `TemplateListResponse`, `TemplateGenerateRequest` |
| `backend/src/pictactic/core/exceptions.py` | Add `TemplateNotFoundError` |
| `backend/src/pictactic/api/app.py` | Import and register templates router |

### Frontend -- New Files

| File | Description |
|---|---|
| `frontend/src/hooks/useTemplates.js` | Templates hook |
| `frontend/src/pages/TemplatesPage.jsx` | Templates page |
| `frontend/src/components/templates/TemplateList.jsx` | Template grid list |
| `frontend/src/components/templates/TemplateCard.jsx` | Single template card |
| `frontend/src/components/templates/TemplateDialog.jsx` | Create/edit modal |
| `frontend/src/components/templates/TemplateFilters.jsx` | Search + category filter |
| `frontend/src/components/templates/SaveAsTemplateButton.jsx` | Save-as-template button |

### Frontend -- Modified Files

| File | Changes |
|---|---|
| `frontend/src/lib/api.js` | Add 6 template API methods to `ApiClient` |
| `frontend/src/main.jsx` | Add `/app/templates` route |
| `frontend/src/components/layout/Header.jsx` | Add templates navigation link |
| `frontend/src/App.jsx` | Import and render `SaveAsTemplateButton` in results area |
| Locale files (en/zh) | Add `templates.*` and `nav.templates` i18n keys |

---

## 9. Implementation Order

For both backend and frontend developers:

### Backend (Task #2)

1. Add `TemplateCategory` enum to `enums.py`
2. Add `TemplateNotFoundError` to `exceptions.py`
3. Add Pydantic schemas to `schemas.py`
4. Add `PromptTemplate` ORM model to `models.py` + `User.templates` relationship
5. Create `template_repository.py`
6. Create `template_service.py`
7. Create `routes/templates.py`
8. Register router in `app.py`

### Frontend (Task #3)

1. Add API client methods to `api.js`
2. Create `useTemplates` hook
3. Create `TemplateFilters` component
4. Create `TemplateCard` component
5. Create `TemplateList` component
6. Create `TemplateDialog` component
7. Create `TemplatesPage`
8. Add route to `main.jsx`
9. Add navigation link to `Header.jsx`
10. Create `SaveAsTemplateButton` and integrate into `App.jsx`
11. Add i18n keys
