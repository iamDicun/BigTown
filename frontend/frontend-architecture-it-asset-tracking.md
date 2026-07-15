# Frontend Architecture — IT Asset & Hardware Tracking System

## 1. Project Context

**Option B — IT Asset & Hardware Tracking System**

### Problem
Tracking monitors, laptops, keyboards, and other company hardware through Excel causes lost assets, unclear responsibility, and poor visibility into who is holding what.

### Core Features
1. **Asset Inventory Dashboard** with real-time statuses:
   - `Available`
   - `Borrowed`
   - `Damaged`
   - `In-Repair`
   - `Lost`
2. **Digital Check-out / Check-in workflow** for employees requesting and returning company equipment.
3. **QR Code identification**: each physical device has a unique QR code for fast scanning, allocation, and return.

---

## 2. Architecture Goal

The frontend is designed to be **clean, modular, and easy to defend during code review**, but it will not copy Backend Clean Architecture one-to-one.

The backend is organized around domain logic, use cases, repositories, and database boundaries. The frontend is organized around:

- user-facing flows,
- screens/pages,
- reusable UI components,
- state management,
- API boundary,
- route/auth flow.

The main goal is:

> The UI should not depend on backend internal implementation. It should only depend on the API contract.

This means the frontend does **not** know whether the backend uses Gin, Clean Architecture, PostgreSQL, raw SQL, GORM, transaction, repository, or service layer. The frontend only knows:

- which API endpoint exists,
- what request payload is required,
- what response shape is returned,
- what HTTP status code means,
- what role is allowed to access which page.

---

## 3. High-Level Frontend Flow

```txt
User interaction
↓
Vue Page/View
↓
Feature Component
↓
Pinia Store / Composable
↓
Feature Service
↓
Shared API Client
↓
Backend API Contract
↓
Response mapped back to frontend state
↓
Vue re-renders UI
```

Example login flow:

```txt
LoginForm.vue
↓ emit("submit", payload)
LoginView.vue
↓ authStore.login(payload)
auth.store.js
↓ authService.login(payload)
auth.service.js
↓ http.post("/auth/login")
shared/api/http.js
↓
Go Backend
↓
Response: token + employee info
↓
authStore saves session state
↓
router redirects to /dashboard
```

Example device checkout flow:

```txt
Employee creates asset request
↓
RequestCreateView.vue
↓
RequestForm.vue emits request payload
↓
requestStore.createRequest(payload)
↓
request.service.js calls POST /requests
↓
Backend creates request + request_detail rows
↓
Frontend refreshes request list and shows status Pending
```

Example IT admin allocation flow:

```txt
IT Admin opens pending request
↓
RequestDetailView.vue
↓
Scans device QR code
↓
requestStore.allocateDevice(requestDetailId, qrCode)
↓
request.service.js calls allocation endpoint
↓
Backend validates device availability
↓
Backend updates request_detail.device_id, allocated_at, status
↓
Backend updates device.current_status = Borrowed
↓
Frontend updates request detail + inventory status
```

---

## 4. Recommended Folder Structure

```txt
src/
├── app/
│   ├── router/
│   │   ├── index.js
│   │   └── guards.js
│   │
│   ├── layouts/
│   │   ├── AppLayout.vue
│   │   ├── AuthLayout.vue
│   │   └── components/
│   │       ├── Navbar.vue
│   │       └── Sidebar.vue
│   │
│   ├── views/
│   │   ├── NotFoundView.vue
│   │   ├── ForbiddenView.vue
│   │   └── ServerErrorView.vue
│   │
│   └── providers/
│       └── pinia.js
│
├── shared/
│   ├── api/
│   │   ├── http.js
│   │   ├── errorMapper.js
│   │   └── tokenStorage.js
│   │
│   ├── components/
│   │   ├── BaseButton.vue
│   │   ├── BaseInput.vue
│   │   ├── BaseSelect.vue
│   │   ├── BaseModal.vue
│   │   ├── BaseTable.vue
│   │   ├── BasePagination.vue
│   │   ├── BaseBadge.vue
│   │   ├── ConfirmDialog.vue
│   │   └── LoadingSpinner.vue
│   │
│   ├── constants/
│   │   ├── roles.js
│   │   ├── deviceStatus.js
│   │   └── requestStatus.js
│   │
│   ├── utils/
│   │   ├── validators.js
│   │   ├── date.js
│   │   └── formatters.js
│   │
│   └── composables/
│       ├── useDebounce.js
│       ├── usePagination.js
│       └── useAsyncState.js
│
├── features/
│   ├── auth/
│   │   ├── views/
│   │   │   └── LoginView.vue
│   │   ├── components/
│   │   │   └── LoginForm.vue
│   │   ├── stores/
│   │   │   └── auth.store.js
│   │   ├── services/
│   │   │   └── auth.service.js
│   │   └── routes.js
│   │
│   ├── dashboard/
│   │   ├── views/
│   │   │   └── DashboardView.vue
│   │   ├── components/
│   │   │   ├── InventorySummaryCard.vue
│   │   │   ├── DeviceStatusChart.vue
│   │   │   └── RecentActivityList.vue
│   │   ├── stores/
│   │   │   └── dashboard.store.js
│   │   ├── services/
│   │   │   └── dashboard.service.js
│   │   └── routes.js
│   │
│   ├── devices/
│   │   ├── views/
│   │   │   ├── DeviceListView.vue
│   │   │   ├── DeviceDetailView.vue
│   │   │   ├── DeviceCreateView.vue
│   │   │   └── DeviceEditView.vue
│   │   ├── components/
│   │   │   ├── DeviceTable.vue
│   │   │   ├── DeviceForm.vue
│   │   │   ├── DeviceStatusBadge.vue
│   │   │   ├── DeviceQRCode.vue
│   │   │   └── DeviceFilterBar.vue
│   │   ├── stores/
│   │   │   └── device.store.js
│   │   ├── services/
│   │   │   └── device.service.js
│   │   ├── composables/
│   │   │   └── useDeviceFilters.js
│   │   └── routes.js
│   │
│   ├── device-types/
│   │   ├── views/
│   │   │   ├── DeviceTypeListView.vue
│   │   │   └── DeviceTypeDetailView.vue
│   │   ├── components/
│   │   │   ├── DeviceTypeTable.vue
│   │   │   └── DeviceTypeForm.vue
│   │   ├── stores/
│   │   │   └── deviceType.store.js
│   │   ├── services/
│   │   │   └── deviceType.service.js
│   │   └── routes.js
│   │
│   ├── requests/
│   │   ├── views/
│   │   │   ├── RequestListView.vue
│   │   │   ├── RequestCreateView.vue
│   │   │   ├── RequestDetailView.vue
│   │   │   └── RequestApprovalView.vue
│   │   ├── components/
│   │   │   ├── RequestTable.vue
│   │   │   ├── RequestForm.vue
│   │   │   ├── RequestStatusBadge.vue
│   │   │   ├── RequestDetailTable.vue
│   │   │   └── AllocationScanner.vue
│   │   ├── stores/
│   │   │   └── request.store.js
│   │   ├── services/
│   │   │   └── request.service.js
│   │   ├── composables/
│   │   │   └── useAllocationScanner.js
│   │   └── routes.js
│   │
│   ├── employees/
│   │   ├── views/
│   │   │   ├── EmployeeListView.vue
│   │   │   └── EmployeeDetailView.vue
│   │   ├── components/
│   │   │   ├── EmployeeTable.vue
│   │   │   ├── EmployeeRoleBadge.vue
│   │   │   └── EmployeeHoldingAssets.vue
│   │   ├── stores/
│   │   │   └── employee.store.js
│   │   ├── services/
│   │   │   └── employee.service.js
│   │   └── routes.js
│   │
│   ├── departments/
│   │   ├── views/
│   │   │   └── DepartmentListView.vue
│   │   ├── components/
│   │   │   └── DepartmentTable.vue
│   │   ├── stores/
│   │   │   └── department.store.js
│   │   ├── services/
│   │   │   └── department.service.js
│   │   └── routes.js
│   │
│   ├── notifications/
│   │   ├── views/
│   │   │   └── NotificationListView.vue
│   │   ├── components/
│   │   │   ├── NotificationDropdown.vue
│   │   │   └── NotificationItem.vue
│   │   ├── stores/
│   │   │   └── notification.store.js
│   │   ├── services/
│   │   │   └── notification.service.js
│   │   └── routes.js
│   │
│   └── device-history/
│       ├── views/
│       │   └── DeviceHistoryView.vue
│       ├── components/
│       │   ├── DeviceHistoryTable.vue
│       │   └── DeviceIncidentForm.vue
│       ├── stores/
│       │   └── deviceHistory.store.js
│       ├── services/
│       │   └── deviceHistory.service.js
│       └── routes.js
│
├── App.vue
└── main.js
```

This structure is feature-based, but still lightweight. It avoids forcing frontend into backend-style Clean Architecture with usecase/repository/interface layers.

---

## 5. Boundary Rules

### 5.1 `app/` Boundary

`app/` contains things that belong to the whole application.

Examples:

- router configuration,
- navigation guards,
- layout components,
- app-level views such as 404, 403, 500,
- global providers such as Pinia setup.

Good examples:

```txt
app/router/index.js
app/router/guards.js
app/layouts/AppLayout.vue
app/layouts/components/Navbar.vue
app/views/NotFoundView.vue
```

`Navbar.vue` and `Sidebar.vue` belong here because they are not owned by `devices`, `requests`, or `employees`. They are part of the application shell.

---

#### 5.1.1 Route Splitting Rule

When the app grows, `app/router/index.js` should not contain every route definition directly.

Recommended approach:

```txt
features/auth/routes.js
features/devices/routes.js
features/requests/routes.js
features/employees/routes.js
```

Then `app/router/index.js` only composes feature routes:

```js
import authRoutes from '@/features/auth/routes'
import deviceRoutes from '@/features/devices/routes'
import requestRoutes from '@/features/requests/routes'

const routes = [
  ...authRoutes,
  ...deviceRoutes,
  ...requestRoutes,
  {
    path: '/:pathMatch(.*)*',
    component: () => import('@/app/views/NotFoundView.vue')
  }
]
```

Rules:

- Feature route files define routes for their own pages only.
- App router owns global route creation, history mode, and global guards.
- Route guards stay in `app/router/guards.js` because auth/role checks are application-level behavior.
- Route meta should be declared near the feature route so permission is visible when reading the feature.

Example feature route:

```js
export default [
  {
    path: '/devices',
    component: () => import('./views/DeviceListView.vue'),
    meta: {
      requiresAuth: true,
      roles: ['IT_Admin']
    }
  }
]
```

---

### 5.2 `shared/` Boundary

`shared/` contains generic, reusable code that does not know about a specific business feature.

Examples:

- base UI components,
- API client,
- error mapper,
- token storage helper,
- validators,
- date/format helpers,
- constants reused across multiple features.

Good examples:

```txt
shared/components/BaseButton.vue
shared/components/BaseInput.vue
shared/components/ConfirmDialog.vue
shared/api/http.js
shared/api/errorMapper.js
shared/utils/validators.js
```

A shared component must not know about domain-specific fields such as:

```txt
employee.full_name
device.current_status
request.expected_return_date
request_detail.allocated_at
```

If a component knows about these fields, it belongs to a feature.

---

### 5.3 `features/` Boundary

`features/` contains code grouped by user-facing capability or business flow.

A feature usually contains:

```txt
views/
components/
stores/
services/
```

Meaning:

- `views/`: route-level pages,
- `components/`: UI pieces specific to this feature,
- `stores/`: Pinia store for feature state,
- `services/`: API functions for this feature.

Example:

```txt
features/devices/views/DeviceListView.vue
features/devices/components/DeviceTable.vue
features/devices/stores/device.store.js
features/devices/services/device.service.js
```

The `devices` feature is allowed to know about:

- `device.id`,
- `device.qr_code`,
- `device.serial_number`,
- `device.current_status`,
- `device.purchase_date`,
- `device.warranty_expiry_date`,
- `device.repair_count`.

---

### 5.4 `composables/` Boundary

Composables contain reusable reactive UI logic. They should not become a hidden service layer.

Use `shared/composables` for generic UI logic that has no business knowledge.

Examples:

```txt
shared/composables/useDebounce.js
shared/composables/usePagination.js
shared/composables/useAsyncState.js
```

Use `features/<feature>/composables` for reactive logic that belongs to one business feature.

Examples:

```txt
features/devices/composables/useDeviceFilters.js
features/requests/composables/useAllocationScanner.js
```

Rules:

- Composables can manage local reactive state, computed values, watchers, browser APIs, and UI interactions.
- Generic composables in `shared` must not know fields such as `device.current_status` or `request.status`.
- Feature composables may know feature fields, but should stay inside that feature.
- Composables should not replace Pinia stores for shared application state.
- Composables should not call Axios directly. If they need data, they should call a store action or receive data/functions as parameters.

Good:

```txt
DeviceListView.vue -> useDeviceFilters(devices) -> filteredDevices
RequestDetailView.vue -> useAllocationScanner({ onScan })
```

Avoid:

```txt
useDeviceFilters.js -> axios.get('/devices')
```

---

## 6. Page Placement Rules

### 6.1 Business Page

If a page is clearly about one business capability, put it inside that feature.

Examples:

```txt
/devices                    → features/devices/views/DeviceListView.vue
/devices/:id                → features/devices/views/DeviceDetailView.vue
/requests                   → features/requests/views/RequestListView.vue
/requests/create            → features/requests/views/RequestCreateView.vue
/employees                  → features/employees/views/EmployeeListView.vue
```

---

### 6.2 Composite Page

If a page combines many backend modules but represents one user-facing capability, create a frontend feature for that capability.

Example:

```txt
/dashboard → features/dashboard/views/DashboardView.vue
```

The dashboard may call APIs related to:

- devices,
- device types,
- requests,
- employees,
- device history.

Even though it consumes many backend modules, it is still a frontend feature because the user-facing flow is “view inventory overview”.

---

### 6.3 App-Level Page

If a page is not business-specific, put it in `app/views`.

Examples:

```txt
/404 → app/views/NotFoundView.vue
/403 → app/views/ForbiddenView.vue
/500 → app/views/ServerErrorView.vue
```

---

## 7. Component Placement Rules

### 7.1 Component Used Only Inside One Page

If a component is only used inside one page, keep it local to that feature.

Example:

```txt
features/requests/components/AllocationScanner.vue
```

Even if `AllocationScanner.vue` is rendered many times inside `RequestDetailView.vue`, it is not automatically shared. Reuse inside one page does not mean global reuse.

### 7.2 Component Used Across Multiple Pages of One Feature

Put it in:

```txt
features/<feature>/components/
```

Example:

```txt
features/devices/components/DeviceStatusBadge.vue
```

It can be used by:

- `DeviceListView.vue`,
- `DeviceDetailView.vue`,
- `DashboardView.vue` if intentionally imported.

However, since it knows about `device_status`, it still belongs to the devices feature unless generalized.

### 7.3 Component Used Across Multiple Features Without Business Knowledge

Put it in:

```txt
shared/components/
```

Example:

```txt
shared/components/BaseBadge.vue
```

Then feature-specific components can wrap it:

```txt
features/devices/components/DeviceStatusBadge.vue
features/requests/components/RequestStatusBadge.vue
```

Both may internally use `BaseBadge.vue`, but they map domain statuses differently.

### 7.4 Layout Component

Put layout-level components in:

```txt
app/layouts/components/
```

Example:

```txt
app/layouts/components/Navbar.vue
app/layouts/components/Sidebar.vue
```

---

## 8. Local → Feature → Shared Rule

Use this rule when unsure:

```txt
Used in one page only
→ keep near that page or inside that feature.

Used in many pages of the same feature
→ features/<feature>/components/.

Used in many features and does not contain business logic
→ shared/components/.

Used as application shell
→ app/layouts/components/.
```

Important rule:

> A component rendered many times inside one page is not automatically shared.

Shared means it is reused across multiple features and does not contain business-specific knowledge.

---

## 9. Frontend Feature vs Backend Module Mapping

Backend modules and frontend features should be aligned in business language, but they do not need to mirror each other one-to-one.

### Backend Module

Backend modules are organized around business logic and data ownership.

Based on the current schema, possible backend modules are:

```txt
auth
employees
departments
device-types
devices
requests
notifications
device-history
```

### Frontend Feature

Frontend features are organized around user-facing screens and workflows.

Recommended frontend features:

```txt
auth
dashboard
employees
departments
device-types
devices
requests
notifications
device-history
```

### Mapping Table

| Backend Area | Database Tables | Frontend Feature | Mapping Type |
|---|---|---|---|
| Auth | `authen`, `refresh_token`, `token_blacklist`, `employee` | `features/auth` | Mostly 1-1 |
| Employees | `employee`, `department` | `features/employees` | Mostly 1-1 |
| Departments | `department`, `employee` | `features/departments` | Mostly 1-1 |
| Device Types | `device_type` | `features/device-types` | 1-1 |
| Devices | `device`, `device_type` | `features/devices` | Mostly 1-1 |
| Requests | `request`, `request_detail`, `device`, `device_type` | `features/requests` | Composite |
| Notifications | `notification` | `features/notifications` | 1-1 or app-level dropdown |
| Device History | `device_history`, `device` | `features/device-history` | Mostly 1-1 |
| Dashboard | Multiple tables | `features/dashboard` | Frontend-only composite feature |

Conclusion:

> Frontend features should align with backend modules when they represent the same business capability, but the mapping is not forced to be one-to-one. Backend modules are organized around domain logic, while frontend features are organized around user flows and screens.

---

## 10. Data Model Boundary

The frontend should not blindly expose database shape to every component.

The database has tables such as:

- `employee`,
- `department`,
- `authen`,
- `refresh_token`,
- `token_blacklist`,
- `notification`,
- `device_type`,
- `device`,
- `request`,
- `request_detail`,
- `device_history`.

The frontend should work with frontend-friendly models returned from services or stores.

Example backend response may use database-style fields:

```json
{
  "id": "...",
  "qr_code": "LAPTOP-001",
  "serial_number": "SN123456",
  "current_status": "Available",
  "purchase_date": "2026-07-01"
}
```

Frontend service can map this to:

```js
{
  id: "...",
  qrCode: "LAPTOP-001",
  serialNumber: "SN123456",
  status: "Available",
  purchaseDate: "2026-07-01"
}
```

This prevents Vue components from depending too tightly on database naming.

Recommended rule:

```txt
API response DTO
↓ mapped inside service
Frontend model used by store/component
```

---

## 11. Service Layer Boundary

Services are the only feature-level files that should call the shared API client directly.

Example:

```txt
features/devices/services/device.service.js
```

Responsible for:

- calling device-related API endpoints,
- mapping backend DTO to frontend model,
- mapping frontend form model to request payload,
- hiding API endpoint details from views/components.

Example responsibilities:

```txt
device.service.js
- getDevices()
- getDeviceById(id)
- createDevice(payload)
- updateDevice(id, payload)
- markDeviceDamaged(id, payload)
- getDeviceByQrCode(qrCode)
```

Components must not call Axios directly.

Bad:

```txt
DeviceTable.vue → axios.get('/devices')
```

Good:

```txt
DeviceListView.vue
↓
deviceStore.fetchDevices()
↓
deviceService.getDevices()
↓
http.get('/devices')
```

---

## 12. Store Boundary

Pinia stores manage state and actions for each feature.

Stores should contain:

- list state,
- selected item state,
- loading state,
- error state,
- actions that call services,
- simple getters.

Example device store state:

```js
{
  devices: [],
  selectedDevice: null,
  loading: false,
  error: null
}
```

Example request store state:

```js
{
  requests: [],
  selectedRequest: null,
  pendingRequests: [],
  loading: false,
  error: null
}
```

Stores should not know low-level Axios configuration.

Bad:

```txt
store manually sets Authorization header for every request
```

Good:

```txt
shared/api/http.js handles token attaching through interceptor
```

---

## 13. API Client Boundary

`shared/api/http.js` is the only place that configures HTTP-level details.

Responsibilities:

- `baseURL`,
- request interceptor,
- attaching access token,
- response interceptor,
- handling 401 globally,
- mapping network errors,
- reading environment variables.

Example environment variable:

```txt
VITE_API_URL=http://localhost:8080/api
```

Use `/api/v1` only if the backend officially adds API versioning.

The API client should attach token automatically:

```txt
Authorization: Bearer <access_token>
```

The API client may handle:

| Status Code | Meaning in Frontend |
|---|---|
| 400 | Bad request / invalid payload |
| 401 | Not authenticated / token invalid or expired |
| 403 | Authenticated but not allowed |
| 404 | Resource not found |
| 409 | Conflict, such as duplicated email or unavailable device |
| 422 | Validation error |
| 500 | Server error |

---

### 13.1 API Contract & Response Envelope

The frontend should depend on the backend API contract, not on backend internal code.

For the current backend, successful responses should be treated as an envelope:

```json
{
  "success": true,
  "data": {}
}
```

List response example:

```json
{
  "success": true,
  "data": [
    {
      "id": "...",
      "qr_code": "LAPTOP-001",
      "current_status": "Available"
    }
  ]
}
```

Error response should also be normalized before reaching components:

```json
{
  "success": false,
  "message": "Không tìm thấy thiết bị",
  "code": "NOT_FOUND"
}
```

Recommended rule:

```txt
Backend response envelope
↓ unwrapped/normalized by shared/api/http.js or feature service
Frontend model
↓ used by store/component
```

`shared/api/http.js` may normalize technical HTTP behavior, but feature services should still own feature model mapping.

Example:

```js
// shared/api/http.js
const response = await axiosInstance.get('/devices/available')
return response.data.data
```

Then service maps DTO to frontend model:

```js
// features/devices/services/device.service.js
export async function getAvailableDevices() {
  const devices = await http.get('/devices/available')
  return devices.map(mapDeviceDtoToModel)
}
```

Rules:

- Components should not read `response.data.data` directly.
- Stores should receive frontend-friendly models from services.
- Services should map snake_case API fields to frontend naming if the team chooses camelCase in Vue.
- API endpoint paths should be centralized inside services, not scattered in components.
- If backend changes envelope shape later, fix it in `http.js` and services, not every component.

---

## 14. Auth and Role Boundary

The schema defines employee roles:

```sql
CREATE TYPE employee_role AS ENUM ('Staff', 'Manager', 'IT_Admin');
```

Frontend roles should map to route permissions.

Recommended role meaning:

| Role | Frontend Permission |
|---|---|
| `Staff` | Create asset requests, view own requests, view assigned assets |
| `Manager` | Approve/reject requests from employees |
| `IT_Admin` | Manage inventory, allocate devices, scan QR codes, update device status |

Route meta example:

```js
{
  path: '/devices',
  component: () => import('@/features/devices/views/DeviceListView.vue'),
  meta: {
    requiresAuth: true,
    roles: ['IT_Admin']
  }
}
```

Guard responsibility:

```txt
If route requires auth and user is not logged in
→ redirect to /login.

If route requires role and user role is not allowed
→ redirect to /403.
```

Important security rule:

> Frontend route guard is only for user experience. Backend middleware must still verify JWT and RBAC for every protected API.

---

### 14.1 Token Storage & Refresh Flow

Current backend auth endpoints:

```txt
POST /auth/login
POST /auth/register
POST /auth/refresh
POST /auth/logout
```

Recommended token strategy for this SPA:

```txt
auth.service.js calls login
↓
auth.store.js stores authenticated user/session state
↓
shared/api/tokenStorage.js stores accessToken and refreshToken
↓
shared/api/http.js attaches accessToken to protected requests
```

Important rule:

> Components and feature stores should not read/write `localStorage` directly. Only `shared/api/tokenStorage.js` should know where tokens are stored.

Simple `tokenStorage.js` responsibility:

```txt
getAccessToken()
getRefreshToken()
setTokens(accessToken, refreshToken)
clearTokens()
```

Access token request flow:

```txt
Feature service calls http.get/post
↓
http.js reads access token from tokenStorage
↓
http.js attaches Authorization: Bearer <access_token>
↓
Backend validates JWT
```

Refresh flow when access token expires:

```txt
API request returns 401
↓
http.js checks refresh token
↓
http.js calls POST /auth/refresh
↓
Backend returns new accessToken + refreshToken
↓
tokenStorage updates tokens
↓
http.js retries the original failed request once
```

If refresh fails:

```txt
clear tokens
clear auth store session
redirect to /login
```

Concurrency rule:

- If many requests fail with `401` at the same time, only one refresh request should run.
- Other failed requests should wait for the same refresh promise, then retry after tokens are updated.

Conceptual example:

```js
let refreshPromise = null

async function refreshTokensOnce() {
  if (!refreshPromise) {
    refreshPromise = authService.refreshToken()
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
}
```

Logout flow:

```txt
auth.store.logout()
↓
auth.service.logout(refreshToken)
↓
backend revokes refresh token and blacklists access token
↓
tokenStorage.clearTokens()
↓
router redirects to /login
```

Security note:

- `localStorage` is simple and works for this project, but it is exposed to XSS.
- Keeping token access behind `tokenStorage.js` makes it easier to migrate later to `sessionStorage`, memory-only access token, or httpOnly cookie strategy.
- Backend remains the source of truth for token validity and permissions.

---

## 15. Feature Responsibilities

### 15.1 `auth`

Responsible for:

- login,
- logout,
- storing authenticated employee profile,
- coordinating session state while actual token persistence stays behind `shared/api/tokenStorage.js`,
- exposing `isAuthenticated`,
- exposing current role.

Related tables:

- `authen`,
- `employee`,
- `refresh_token`,
- `token_blacklist`.

Possible pages/components:

```txt
LoginView.vue
LoginForm.vue
```

---

### 15.2 `dashboard`

Responsible for inventory overview.

It may show:

- total device types,
- total physical devices,
- available devices,
- borrowed devices,
- damaged devices,
- in-repair devices,
- lost devices,
- pending requests,
- recent incidents.

Related tables:

- `device_type`,
- `device`,
- `request`,
- `request_detail`,
- `device_history`.

This is a composite frontend feature.

---

### 15.3 `devices`

Responsible for physical device management.

Related table:

- `device`.

Important fields:

- `qr_code`,
- `serial_number`,
- `current_status`,
- `purchase_date`,
- `warranty_expiry_date`,
- `repair_count`.

Possible UI:

- device list,
- device detail,
- QR code display,
- device status badge,
- device filter by status,
- create/edit device form.

---

### 15.4 `device-types`

Responsible for device catalog management.

Related table:

- `device_type`.

Important fields:

- `name`,
- `category`,
- `brand`,
- `specifications`,
- `base_price`,
- `warranty_duration_months`,
- `total_quantity`,
- `available_quantity`.

Possible UI:

- device type list,
- device type detail,
- device type form.

---

### 15.5 `requests`

Responsible for asset request, approval, allocation, and return flow.

Related tables:

- `request`,
- `request_detail`,
- `device`,
- `device_type`.

Important statuses:

```txt
request_status: Pending, Approved, Rejected, Cancelled, Completed
item_status: Pending, Allocated, Returned
```

Possible UI:

- request list,
- create request form,
- request detail,
- approval page,
- allocation scanner,
- check-in / return workflow.

This feature is composite because it coordinates request header, request detail, device type, and physical device allocation.

---

### 15.6 `employees`

Responsible for employee profile and employee asset holding visibility.

Related tables:

- `employee`,
- `department`,
- `request`,
- `request_detail`.

Important fields:

- `full_name`,
- `email`,
- `role`,
- `current_holding_assets`.

Possible UI:

- employee list,
- employee detail,
- current borrowed assets.

---

### 15.7 `departments`

Responsible for department information.

Related table:

- `department`.

Important fields:

- `name`,
- `manager_id`.

Possible UI:

- department list,
- department detail,
- manager display.

---

### 15.8 `notifications`

Responsible for displaying notifications.

Related table:

- `notification`.

Possible UI:

- notification dropdown in navbar,
- notification list page,
- mark as read.

This feature can be used inside `app/layouts/components/Navbar.vue`, but its state and API should still live in `features/notifications`.

---

### 15.9 `device-history`

Responsible for incident and maintenance records.

Related table:

- `device_history`.

Important event types:

```txt
Maintenance
Repair
Lost
```

Possible UI:

- device history table,
- incident report form,
- repair cost display,
- history attached to device detail page.

---

## 16. Realtime Boundary

The core feature requires real-time visibility of statuses.

Realtime should be isolated behind a composable or service instead of being called directly by every component.

Recommended location:

```txt
shared/realtime/realtimeClient.js
```

or feature-specific:

```txt
features/devices/composables/useDeviceStatusStream.js
```

Use feature-specific composable if the stream only serves devices.

Use shared realtime client if many features need the same WebSocket/SSE connection.

Example flow:

```txt
Backend emits device status update
↓
realtime client receives event
↓
deviceStore updates device.current_status
↓
Dashboard and DeviceList re-render
```

Rule:

> Components should not manually manage WebSocket connection details. They should consume state from store or composables.

---

## 17. QR Code Boundary

QR code is a core business feature for physical device identification.

Recommended placement:

```txt
features/devices/components/DeviceQRCode.vue
```

If QR scanning is used for allocation and return workflow, scanner UI may live in:

```txt
features/requests/components/AllocationScanner.vue
```

Reason:

- `DeviceQRCode.vue` displays or generates QR for a device.
- `AllocationScanner.vue` belongs to the request allocation/check-in workflow.

If a generic QR scanner component is later needed by multiple features, extract the low-level camera/scanner into:

```txt
shared/components/QrScanner.vue
```

Then feature components wrap it:

```txt
features/requests/components/AllocationScanner.vue
features/devices/components/DeviceLookupScanner.vue
```

---

## 18. Form Model Boundary

Frontend form models do not need to match database tables exactly.

Example request creation form:

```js
{
  expectedReturnDate: '2026-07-30',
  items: [
    {
      deviceTypeId: '...',
      quantity: 2
    }
  ]
}
```

Backend may create:

- one `request` row,
- many `request_detail` rows.

The frontend form should represent user intent, not database implementation.

Rule:

```txt
Form model = what user inputs.
Request DTO = what backend endpoint expects.
Database schema = backend internal storage.
```

Service layer maps form model to request DTO.

---

## 19. Validation Boundary

Frontend validation improves user experience, but it is not security.

Frontend should validate:

- required fields,
- email format,
- positive quantity,
- expected return date is not in the past,
- device type is selected,
- QR code is not empty when scanning.

Backend must still validate everything again.

Important rule:

> Never trust frontend validation. Backend remains the source of truth.

---

## 20. Naming Conventions

### View/Page

Use suffix `View`:

```txt
DeviceListView.vue
RequestDetailView.vue
LoginView.vue
```

### Feature Component

Use business name:

```txt
DeviceTable.vue
RequestForm.vue
EmployeeRoleBadge.vue
```

### Shared Component

Use `Base` prefix for generic UI:

```txt
BaseButton.vue
BaseInput.vue
BaseTable.vue
BaseBadge.vue
```

### Store

Use feature name:

```txt
auth.store.js
device.store.js
request.store.js
```

### Service

Use feature name:

```txt
auth.service.js
device.service.js
request.service.js
```

---

## 21. Dependency Rules

Allowed dependency direction:

```txt
features → shared
app → features
app → shared
```

Feature-to-feature imports should be avoided unless there is a clear reason.

Preferred:

```txt
features/dashboard/services/dashboard.service.js calls dashboard summary API
```

Instead of:

```txt
dashboard imports stores from devices, requests, employees directly
```

If multiple features need the same generic logic, move that logic to `shared`.

---

## 22. What Not To Do

### Do not call Axios directly inside components

Bad:

```txt
DeviceTable.vue calls axios.get('/devices')
```

Good:

```txt
DeviceListView.vue → deviceStore → deviceService → http client
```

### Do not put business components into shared too early

Bad:

```txt
shared/components/DeviceStatusBadge.vue
shared/components/RequestStatusBadge.vue
```

Good:

```txt
features/devices/components/DeviceStatusBadge.vue
features/requests/components/RequestStatusBadge.vue
shared/components/BaseBadge.vue
```

### Do not mirror backend database schema blindly in frontend state

Bad:

```txt
Every component directly uses raw database field names and response shape.
```

Good:

```txt
Service maps API DTO into frontend model.
```

### Do not force backend Clean Architecture into frontend

Bad:

```txt
frontend/domain
frontend/usecases
frontend/repositories
frontend/adapters
```

Good for this project:

```txt
app
shared
features
```

---

## 23. Simple Implementation Steps

Build the frontend in this order:

1. Create base Vue project structure: `app`, `shared`, `features`.
2. Create `shared/api/http.js`, `shared/api/tokenStorage.js`, and `shared/api/errorMapper.js` first.
3. Implement `features/auth`: `auth.service.js`, `auth.store.js`, `LoginView.vue`, `LoginForm.vue`, and `features/auth/routes.js`.
4. Implement router composition in `app/router/index.js` and global guards in `app/router/guards.js`.
5. Add layouts: `AuthLayout.vue`, `AppLayout.vue`, `Navbar.vue`, `Sidebar.vue`.
6. Implement one simple protected feature first, preferably `devices`.
7. For each feature, create in this order: `routes.js`, `service`, `store`, `views`, then components.
8. Keep API paths inside services only.
9. Keep token handling inside `tokenStorage.js` and `http.js` only.
10. Extract shared UI only after at least two features need the same generic component.
11. Add feature composables only when reactive UI logic starts repeating or becomes too large for a view.
12. After one feature works end-to-end, repeat the same pattern for `requests`, `employees`, `notifications`, and dashboard.

Simple rule while coding:

```txt
Page/View -> Store -> Service -> HTTP Client -> Backend API
```

If a file breaks this flow, check whether it is doing too much.

---

## 24. Final Architecture Summary

The frontend architecture for this project is:

```txt
app/
→ application shell, routing, layouts, guards, global pages

shared/
→ generic reusable UI, API client, utilities, constants

features/
→ business/user-facing flows such as auth, dashboard, devices, requests, employees
```

The most important boundaries are:

```txt
Component does not call API directly.
View coordinates screen-level behavior.
Store manages state.
Service talks to API and maps DTOs.
Shared API client handles HTTP technical details.
Route guard handles UX-level auth and role checks.
Backend still enforces real security.
```

Frontend features should align with backend modules by business language, but should not be forced to mirror backend one-to-one.

The frontend depends on the API contract, not backend implementation.
