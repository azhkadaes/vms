import { Loading } from "@vms/ui";
import { lazy, Suspense } from "react";

const Employees = lazy(() => import("../features/EmployeeDirectory"));
const Roles = lazy(() => import("../features/RoleManagement"));

export default function Settings({ page }: { page: string }) {
  return (
    <Suspense fallback={<Loading />}>
      {page === "employees" ? <Employees /> : page === "roles" ? <Roles /> : null}
    </Suspense>
  );
}
