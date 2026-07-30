import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import BlogUsersPage from "./BlogUsersPage";

vi.mock("../api/admin", () => ({
  listBlogUsers: vi.fn(),
  createBlogUser: vi.fn(),
  updateBlogUser: vi.fn(),
  deleteBlogUser: vi.fn(),
  resetBlogUserPassword: vi.fn(),
}));

import * as adminApi from "../api/admin";

const user = (o: Partial<import("../api/admin").BlogUser> = {}) => ({
  id: "1",
  username: "alice",
  isEnabled: true,
  tokenVersion: 0,
  deletedAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
  ...o,
});

beforeEach(() => {
  vi.clearAllMocks();
  (adminApi.listBlogUsers as ReturnType<typeof vi.fn>).mockResolvedValue([user()]);
  (adminApi.createBlogUser as ReturnType<typeof vi.fn>).mockResolvedValue(user({ username: "new" }));
  (adminApi.updateBlogUser as ReturnType<typeof vi.fn>).mockResolvedValue(user({ isEnabled: false }));
  (adminApi.deleteBlogUser as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);
  (adminApi.resetBlogUserPassword as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);
  // listBlogUsers 在 create/toggle/delete 后被 reload 再次调用，默认返回空列表便于断言
});

describe("BlogUsersPage", () => {
  it("加载并展示博客账号列表", async () => {
    render(<BlogUsersPage />);
    expect(await screen.findByText("alice")).toBeInTheDocument();
    expect(adminApi.listBlogUsers).toHaveBeenCalledTimes(1);
  });

  it("创建账号", async () => {
    (adminApi.listBlogUsers as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce([user()])
      .mockResolvedValueOnce([]);
    render(<BlogUsersPage />);
    await screen.findByText("alice");
    await userEvent.type(screen.getByPlaceholderText("用户名（≥3）"), "bob");
    await userEvent.type(screen.getByPlaceholderText("密码（≥6）"), "secret1");
    await userEvent.click(screen.getByRole("button", { name: "创建" }));
    await waitFor(() => expect(adminApi.createBlogUser).toHaveBeenCalledWith({ username: "bob", password: "secret1" }));
  });

  it("停用账号", async () => {
    render(<BlogUsersPage />);
    await screen.findByText("alice");
    await userEvent.click(screen.getByRole("button", { name: "停用" }));
    await waitFor(() =>
      expect(adminApi.updateBlogUser).toHaveBeenCalledWith("1", { isEnabled: false }),
    );
  });

  it("重置密码", async () => {
    render(<BlogUsersPage />);
    await screen.findByText("alice");
    await userEvent.click(screen.getByRole("button", { name: "重置密码" }));
    const modal = await screen.findByText(/重置「alice」密码/);
    const input = within(modal.parentElement!).getByPlaceholderText("新密码（≥6）");
    await userEvent.type(input, "newpass1");
    await userEvent.click(within(modal.parentElement!).getByRole("button", { name: "重置" }));
    await waitFor(() => expect(adminApi.resetBlogUserPassword).toHaveBeenCalledWith("1", "newpass1"));
  });

  it("软删除账号（确认后）", async () => {
    const spy = vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<BlogUsersPage />);
    await screen.findByText("alice");
    await userEvent.click(screen.getByRole("button", { name: "删除" }));
    await waitFor(() => expect(adminApi.deleteBlogUser).toHaveBeenCalledWith("1"));
    spy.mockRestore();
  });

  it("删除取消则不调用 delete", async () => {
    const spy = vi.spyOn(window, "confirm").mockReturnValue(false);
    render(<BlogUsersPage />);
    await screen.findByText("alice");
    await userEvent.click(screen.getByRole("button", { name: "删除" }));
    await waitFor(() => expect(spy).toHaveBeenCalled());
    expect(adminApi.deleteBlogUser).not.toHaveBeenCalled();
    spy.mockRestore();
  });
});
