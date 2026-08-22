import { expect, test, type Page } from "@playwright/test";

const admin = { username: "admin", password: "Admin@12345" };

async function login(page: Page) {
  await page.goto("/login");
  await page.getByPlaceholder(/用户|账号|username/i).fill(admin.username);
  await page.getByPlaceholder(/密码|password/i).fill(admin.password);
  await page.getByRole("button", { name: /接入|登录|登/ }).click();
  await expect(page).not.toHaveURL(/login/, { timeout: 20_000 });
}

test.describe("GoScrapy critical flow (mock)", () => {
  test("health page shell loads", async ({ page }) => {
    const res = await page.goto("/");
    expect(res?.ok() || page.url().includes("login")).toBeTruthy();
  });

  test("login then visit dashboard / rules / tasks", async ({ page }) => {
    await login(page);
    await page.goto("/dashboard");
    await expect(page.locator("body")).toContainText(/集群|节点|Pages|速率|GOSCRAPY/i);
    await page.goto("/rules");
    await expect(page.locator("body")).toContainText(/规则/);
    await page.goto("/tasks");
    await expect(page.locator("body")).toContainText(/任务/);
  });

  test("create task from seeded rule and see results table", async ({ page }) => {
    await login(page);
    await page.goto("/tasks");
    await page.getByRole("button", { name: "创建任务" }).click();
    await page.getByPlaceholder("demo-crawl").fill("e2e-mock-task");
    await page.locator(".n-modal .n-select").first().click();
    await page.locator(".n-base-select-option").first().click();
    await page.getByRole("button", { name: "提交创建" }).click();
    await expect(page.locator("body")).toContainText(/e2e-mock-task/);
  });
});
