import { apiClient } from './client'

// ==================== Types ====================

export interface Dashboard {
  id: string
  app: string
  title: string
  description: string
  widgetCount: number
  createdAt: string
  updatedAt: string
}

export interface Widget {
  id: string
  dashboardId: string
  type: 'stat' | 'chart' | 'table'
  title: string
  dataSource: string
  config: Record<string, any>
  gridX: number
  gridY: number
  gridW: number
  gridH: number
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface DataSourceMeta {
  key: string
  description: string
}

export interface ChartData {
  label: string
  value: number
  extra?: Record<string, any>
}

// ==================== Dashboard ====================

/** 列出所有仪表盘 */
export async function listDashboards(): Promise<Dashboard[]> {
  const { data } = await apiClient.get<Dashboard[]>('/admin/dashboards')
  return data
}

/** 获取单个仪表盘及其 widgets */
export async function getDashboard(
  id: string
): Promise<{ dashboard: Dashboard; widgets: Widget[] }> {
  const { data } = await apiClient.get<{ dashboard: Dashboard; widgets: Widget[] }>(
    `/admin/dashboards/${id}`
  )
  return data
}

/** 更新仪表盘标题/描述 */
export async function updateDashboard(
  id: string,
  req: { title?: string; description?: string }
): Promise<Dashboard> {
  const { data } = await apiClient.put<Dashboard>(`/admin/dashboards/${id}`, req)
  return data
}

// ==================== Widget ====================

/** 在仪表盘下新建 widget */
export async function createWidget(
  dashboardId: string,
  req: Omit<Widget, 'id' | 'dashboardId' | 'createdAt' | 'updatedAt'>
): Promise<Widget> {
  const { data } = await apiClient.post<Widget>(
    `/admin/dashboards/${dashboardId}/widgets`,
    req
  )
  return data
}

/** 更新 widget */
export async function updateWidget(
  dashboardId: string,
  widgetId: string,
  req: Partial<Widget>
): Promise<Widget> {
  const { data } = await apiClient.put<Widget>(
    `/admin/dashboards/${dashboardId}/widgets/${widgetId}`,
    req
  )
  return data
}

/**
 * 更新 Widget 的网格位置（仅 gridX/gridY/gridW/gridH）。
 * 使用 PATCH 语义，不会覆盖 type/title/dataSource/config。
 */
export async function updateWidgetLayout(
  dashboardId: string,
  widgetId: string,
  req: { gridX?: number; gridY?: number; gridW?: number; gridH?: number }
): Promise<void> {
  await apiClient.patch(
    `/admin/dashboards/${dashboardId}/widgets/${widgetId}/layout`,
    req
  )
}

/** 删除 widget */
export async function deleteWidget(
  dashboardId: string,
  widgetId: string
): Promise<void> {
  await apiClient.delete(`/admin/dashboards/${dashboardId}/widgets/${widgetId}`)
}

// ==================== DataSource ====================

/** 列出可用数据源 */
export async function listDataSources(): Promise<DataSourceMeta[]> {
  const { data } = await apiClient.get<DataSourceMeta[]>('/admin/datasources')
  return data
}

/**
 * 查询数据源。
 * key 使用 `:` 分隔（如 `finflow:transactions`），在 URL path 中以字面 `:` 发送。
 * params 作为 JSON body 发送。
 */
export async function queryDataSource(
  key: string,
  params?: Record<string, any>
): Promise<ChartData[]> {
  const { data } = await apiClient.post<{ key: string; data: ChartData[] }>(
    `/admin/datasources/${key}/query`,
    params ?? {}
  )
  return data.data
}
