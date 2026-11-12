import { Loading } from "@vms/ui";
import { lazy, Suspense } from "react";

const Reports = lazy(() => import("./Reports"));
const Employees = lazy(() => import("../features/EmployeeDirectory"));
const Roles = lazy(() => import("../features/RoleManagement"));
const Integrations = lazy(() => import("../features/IntegrationManagement"));

export default function Settings({ page }: { page: string }) {
  return (
    <Suspense fallback={<Loading />}>
      {page === "analytics" || page === "logs" ? (
        <Reports page={page} />
      ) : page === "employees" ? (
        <Employees />
      ) : page === "roles" ? (
        <Roles />
      ) : (
        <Integrations />
      )}
    </Suspense>
  );
}
