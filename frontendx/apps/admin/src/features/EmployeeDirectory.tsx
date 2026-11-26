import DataRefresh from "../components/DataRefresh";
import { managementFreshness } from "../hooks/managementFreshness";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, preview, sessionKey, useSession } from "@vms/data";
import { management } from "@vms/data/management";
import type { Employee } from "@vms/data/management-contract";
import { Button, Empty, ErrorNotice, Loading, SearchField, TableRegion } from "@vms/ui";
import { Eye, Plus } from "lucide-react";
import { useRef, useState, type MouseEvent } from "react";
import { useSearchParams } from "react-router";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import EmployeeEditor from "./EmployeeEditor";

function DirectoryRecords({
  people,
  onView,
}: {
  people: Employee[];
  onView: (person: Employee, trigger: HTMLElement) => void;
}) {
  function openRecord(person: Employee, event: MouseEvent<HTMLElement>) {
    if (event.target instanceof Element && event.target.closest("button, a")) return;
    const trigger = event.currentTarget.querySelector<HTMLButtonElement>(".employee-name");
    if (trigger) onView(person, trigger);
  }
  const name = (person: Employee) => (
    <Button
      type="button"
      kind="row-link"
      className="employee-name"
      aria-label={`Lihat detail pegawai ${person.name}`}
      onClick={(event) => onView(person, event.currentTarget)}
    >
      {person.name}
    </Button>
  );
  const details = (person: Employee) => (
    <Button
      kind="ghost"
      className="employee-detail-action"
      onClick={(event) => onView(person, event.currentTarget)}
      aria-label={`Detail pegawai ${person.name}`}
      title={`Lihat detail pegawai ${person.name}`}
    >
      <Eye size={16} aria-hidden="true" />
      <span className="employee-detail-label">Detail</span>
    </Button>
  );
  return (
    <>
      <div className="employee-cards" aria-label="Daftar pegawai">
        {people.map((person) => (
          <article
            className="employee-card"
            key={person.id}
            onClick={(event) => openRecord(person, event)}
          >
            <h3>{name(person)}</h3>
            <dl>
              <div>
                <dt>Email</dt>
                <dd>{person.email}</dd>
              </div>
              <div>
                <dt>Peran pegawai</dt>
                <dd>{(person.roles || [person.role]).join(", ")}</dd>
              </div>
              <div>
                <dt>Status akun</dt>
                <dd>{person.active ? "Aktif" : "Nonaktif"}</dd>
              </div>
            </dl>
            {details(person)}
          </article>
        ))}
      </div>
      <TableRegion className="employee-table" label="Direktori pegawai">
        <table>
          <caption className="sr-only">Direktori pegawai sesuai pencarian</caption>
          <thead>
            <tr>
              <th scope="col">Nama</th>
              <th scope="col">Email</th>
              <th scope="col">Peran pegawai</th>
              <th scope="col">Status akun</th>
              <th scope="col" className="employee-actions">
                <span className="sr-only">Tindakan</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {people.map((person) => (
              <tr
                key={person.id}
                className="employee-row"
                onClick={(event) => openRecord(person, event)}
              >
                <td>{name(person)}</td>
                <td>{person.email}</td>
                <td>{(person.roles || [person.role]).join(", ")}</td>
                <td>{person.active ? "Aktif" : "Nonaktif"}</td>
                <td className="employee-actions">{details(person)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableRegion>
    </>
  );
}

export default function EmployeeDirectory() {
  const { session, refreshSession } = useSession();
  const client = useQueryClient();
  const [params, setParams] = useSearchParams();
  const search = params.get("q") || "";
  const settledSearch = useDebouncedValue(search, preview ? 0 : 250);
  const requestedPage = Number(params.get("page"));
  const page = Number.isSafeInteger(requestedPage) && requestedPage > 0 ? requestedPage : 1;
  const query = useQuery({
    ...managementFreshness,
    queryKey: ["employees", sessionKey(session), settledSearch, page],
    queryFn: ({ signal }) => management.employees({ q: settledSearch, page, pageSize: 25 }, signal),
    enabled: !!session,
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[1] === sessionKey(session) ? previous : undefined,
  });
  const [editing, setEditing] = useState<{ employee: Employee | null; key: string } | null>(null);
  const [notice, setNotice] = useState("");
  const trigger = useRef<HTMLElement | null>(null);
  const searchInput = useRef<HTMLInputElement>(null);
  function changeSearch(value: string) {
    const next = new URLSearchParams(params);
    next.delete("page");
    if (value) next.set("q", value);
    else next.delete("q");
    setParams(next, { replace: true });
  }
  function changePage(value: number) {
    const next = new URLSearchParams(params);
    next.set("page", String(value));
    setParams(next);
  }
  const data =
    query.error instanceof ApiError && [401, 403].includes(query.error.status)
      ? undefined
      : query.data;
  return (
    <>
      <div className="page-intro employee-intro">
        <div>
          <h1>Pegawai</h1>
          <p>Direktori dan tanggung jawab pegawai.</p>
        </div>
        <Button
          disabled={!data}
          onClick={(event) => {
            trigger.current = event.currentTarget;
            setEditing({ employee: null, key: crypto.randomUUID() });
          }}
        >
          <Plus size={16} />
          {preview ? "Tambah pegawai demo" : "Tambah pegawai"}
        </Button>
      </div>
      {notice && <p role="status">{notice}</p>}
      <DataRefresh q={query} label="direktori pegawai" />
      <section
        className="panel employee-directory"
        aria-labelledby="employee-directory-title"
        aria-busy={query.isFetching || settledSearch !== search}
      >
        <div className="panel-heading employee-heading">
          <div>
            <h2 id="employee-directory-title">Direktori pegawai</h2>
            {query.isPlaceholderData ? (
              <p role="status">Memperbarui hasil pencarian…</p>
            ) : (
              data && (
                <p className="muted" role="status">
                  Menampilkan {data.items.length} dari {data.totalAll} pegawai.
                </p>
              )
            )}
          </div>
          <div className="employee-search">
            <label htmlFor="employee-search">Cari nama, email, atau peran</label>
            <SearchField
              ref={searchInput}
              id="employee-search"
              placeholder="Cari pegawai…"
              value={search}
              onChange={(event) => changeSearch(event.target.value)}
            />
            {!!search && (
              <Button
                kind="ghost"
                onClick={() => {
                  changeSearch("");
                  searchInput.current?.focus();
                }}
              >
                Reset pencarian
              </Button>
            )}
          </div>
        </div>
        {query.isPending && <Loading />}
        {query.error && (
          <>
            <ErrorNotice error={query.error} />
            <Button kind="secondary" onClick={() => void query.refetch()}>
              Coba lagi
            </Button>
          </>
        )}
        {data &&
          (data.items.length ? (
            <DirectoryRecords
              people={data.items}
              onView={(employee, element) => {
                trigger.current = element;
                setEditing({ employee, key: crypto.randomUUID() });
              }}
            />
          ) : (
            <Empty
              title={data.totalAll ? "Tidak ada pegawai yang sesuai" : "Belum ada pegawai"}
              text={
                data.totalAll
                  ? "Coba nama, email, atau peran lain. Gunakan Reset pencarian untuk melihat semua pegawai."
                  : "Tambahkan pegawai untuk mengisi direktori."
              }
            />
          ))}
        {data && (page > 1 || data.total > data.pageSize) && (
          <nav className="form-actions" aria-label="Halaman direktori">
            <Button kind="secondary" disabled={page <= 1} onClick={() => changePage(page - 1)}>
              Sebelumnya
            </Button>
            <span>
              Halaman {page} dari {Math.max(1, Math.ceil(data.total / data.pageSize))}
            </span>
            <Button
              kind="secondary"
              disabled={page * data.pageSize >= data.total}
              onClick={() => changePage(page + 1)}
            >
              Berikutnya
            </Button>
          </nav>
        )}
        {preview && (
          <p className="hint employee-preview-note">
            Data contoh lokal; tidak membuat akun backend atau menyimpan kata sandi.
          </p>
        )}
      </section>
      {editing &&
        query.data &&
        !(query.error instanceof ApiError && [401, 403].includes(query.error.status)) && (
          <EmployeeEditor
            key={editing.key}
            employee={editing.employee}
            minimumPassword={query.data.passwordMinLength}
            returnFocusTo={trigger.current}
            onClose={() => setEditing(null)}
            onSaved={() => {
              setEditing(null);
              setNotice(preview ? "Contoh pegawai disimpan." : "Pegawai disimpan.");
              void client.invalidateQueries({ queryKey: ["employees"] });
              void client.invalidateQueries({ queryKey: ["employee-detail"] });
              void refreshSession();
            }}
            onReload={async () => {
              if (!editing.employee) return;
              const employee = await management.employee(editing.employee.id);
              setEditing({ employee, key: crypto.randomUUID() });
            }}
          />
        )}
    </>
  );
}
