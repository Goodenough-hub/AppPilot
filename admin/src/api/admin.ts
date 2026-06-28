import { apiClient } from './client'

export interface UserStats {
  transactionCount: number
  lastActiveAt: string | null
}

export interface User {
  id: string
  username: string
  role: string
  appScope: string[]
  createdAt: string
  updatedAt: string
  stats?: UserStats
}

export interface CreateUserRequest {
  username: string
  password: string
  role?: 'user' | 'admin'
  appScope?: string[]
}

export interface Transaction {
  id: string
  amount: string
  type: 'income' | 'expense' | 'transfer'
  note: string
  date: string
  time?: string
  categoryId?: string
  accountId?: string
  toAccountId?: string
  vendor?: string
  createdAt: string
}

export interface Category {
  id: string
  name: string
  type: 'income' | 'expense'
  icon: string
  colorHex: string
  sortOrder: number
  isSystem: boolean
  parentId?: string
}

export interface Account {
  id: string
  name: string
  type: string
  icon: string
  colorHex: string
  initialBalance: string
  sortOrder: number
  isSystem: boolean
}

export interface AdminStats {
  totalUsers: number
  totalTransactions: number
  admins: number
  regularUsers: number
  activeThisWeek?: number
}

export async function listApps(): Promise<string[]> {
  const { data } = await apiClient.get<string[]>('/admin/apps')
  return data
}

export async function listUsers(app?: string): Promise<User[]> {
  const params = app ? { app } : undefined
  const { data } = await apiClient.get<User[]>('/admin/users', { params })
  return data
}

export async function createUser(req: CreateUserRequest): Promise<User> {
  const { data } = await apiClient.post<User>('/admin/users', req)
  return data
}

export async function deleteUser(id: string): Promise<void> {
  await apiClient.delete(`/admin/users/${id}`)
}

export async function getUserTransactions(id: string): Promise<Transaction[]> {
  const { data } = await apiClient.get<Transaction[]>(`/admin/users/${id}/transactions`)
  return data
}

export async function getUserCategories(id: string): Promise<Category[]> {
  const { data } = await apiClient.get<Category[]>(`/admin/users/${id}/categories`)
  return data
}

export async function getUserAccounts(id: string): Promise<Account[]> {
  const { data } = await apiClient.get<Account[]>(`/admin/users/${id}/accounts`)
  return data
}

export async function getStats(app?: string): Promise<AdminStats> {
  const params = app ? { app } : undefined
  const { data } = await apiClient.get<AdminStats>('/admin/stats', { params })
  return data
}
