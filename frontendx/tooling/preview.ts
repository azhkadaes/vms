import type { Plugin } from "vite";
import { readFilter } from "../packages/data/src/report-filter.ts";
import { createHash } from "node:crypto";
import { Readable } from "node:stream";
import {
  aggregateVisits,
  canReadSummary,
  canExportSummary,
  summaryCsv,
  summaryFilterSchema,
  type VisitSummary,
} from "../packages/data/src/reports.ts";
import { hasReservationConflict } from "../packages/data/src/scheduling.ts";
import {
  canReadVisitLog, projectVisitLog, selectVisitLog, summarizeVisitLog, visitLogFilterSchema,
} from "../packages/data/src/visit-log.ts";
import { allowedActions, can } from "./preview-policy.ts";
import {
  canCreateAppointment,
  isRestricted,
  matchesAttachmentSignature,
  notaDinasType,
  officers,
  rooms,
  roomOptions,
  roles,
  scope,
  sessionFor,
  today,
  validateReschedule,
  toVisitorBooking as projectVisitor,
  toStaffBooking as projectStaff,
  visitorPhotoTypes,
  getVisitTiming,
  canVisitorCancel,
  type PublicVisitAction,
  type Booking,
  type StoredBooking,
  type Role,
  type Session,
} from "../packages/data/src/domain.ts";
import {
  rejectionSchema,
  visitNoteFields,
  staffCancellationSchema,
  restrictedAppointmentSchema,
  standardStaffAppointmentInputSchema,
  publicBookingInputSchema,
  publicBookingCreateSchema,
  emailSchema,
} from "../packages/data/src/validation.ts";
export function previewApi({
  now = Date.now,
  session = sessionFor,
}: { now?: () => number; session?: (role: Role) => Session } = {}): Plugin {
  const reservationConflict = { message: "Ruangan atau pejabat memiliki jadwal yang bertabrakan." };
  const visitorPhotos = new Map<string, { bytes: Buffer; type: string }>();
  const documents = new Map<string, { bytes: Buffer; type: string }>();
  const attempts = new Map<
    string,
    { state: "committed"; payload: string; booking: StoredBooking } | { state: "closed" }
  >();
  const staffAttempts = new Map<string, { payload: string; booking: StoredBooking }>();
  const names = [
    "Aditya Pratama",
    "Salsabila Putri",
    "Bima Arya",
    "Nadia Maharani",
    "Rizky Ramadhan",
    "Dewi Lestari",
    "Fajar Nugraha",
    "Citra Anggraini",
    "Arman Setiawan",
    "Indah Permata",
  ];
  let bookings: StoredBooking[] = names.map((name, i) => ({
    id: `demo-${i + 1}`,
    code: `880000${String(i + 1).padStart(2, "0")}`,
    name,
    organization: [
      "Kementerian PUPR",
      "Universitas Mulawarman",
      "PT Bina Karya",
      "Pemerintah Kota Balikpapan",
    ][i % 4],
    email: "visitor@example.test",
    phone: "+62 800 0000 0000",
    officerId: officers[i % 3].id,
    host: officers[i % 3].name,
    vip: i % 3 === 0,
    date: today(),
    time: `${String(9 + Math.floor(i / 2)).padStart(2, "0")}:${i % 2 ? "30" : "00"}`,
    duration: 60,
    room: rooms[i % 3],
    purpose: [
      "Koordinasi pembangunan kawasan",
      "Diskusi kerja sama pendidikan",
      "Audiensi dan koordinasi program",
    ][i % 3],
    visitors: [name, ...(i % 2 ? ["Anggota Rombongan Demo"] : [])],
    identityType: ["KTP", "Paspor", "SIM"][i % 3],
    status: (
      [
        "pending",
        "approved",
        "pending",
        "checked_in",
        "approved",
        "checked_out",
        "pending",
        "rejected",
        "cancelled",
        "approved",
      ] as const
    )[i],
    version: 1,
    rescheduled: i === 4,
    history: [
      { at: new Date(now()).toISOString(), action: "received", message: "Permohonan diterima" },
    ],
  }));
  bookings.push(
    ...(["approved", "approved"] as const).map((status, i) => ({
      id: `demo-restricted-${i + 1}`,
      code: `8800009${i}`,
      visitKind: "restricted" as const,
      name: "",
      organization: "",
      email: "",
      phone: "",
      officerId: i ? "o2" : "",
      host: i ? officers[1].name : "",
      vip: false,
      date: today(),
      time: i ? "15:00" : "14:00",
      duration: 60,
      room: "",
      purpose: "",
      visitors: [],
      identityType: "",
      status,
      version: 1,
      createdBy: { name: "Nadia Putri", role: "Protocol Officer" as const },
      history: [
        {
          at: new Date(now()).toISOString(),
          action: "received",
          message: "Contoh kunjungan terbatas untuk pratinjau lokal.",
        },
      ],
    })),
  );
  return {
    name: "local-preview-only",
    apply: "serve",
    configureServer(server) {
      server.middlewares.use("/__preview", async (req, res, next) => {
        if (!["127.0.0.1", "::1", "::ffff:127.0.0.1"].includes(req.socket.remoteAddress || "")) {
          res.writeHead(403).end();
          return;
        }
        res.setHeader("Cache-Control", "no-store");
        res.setHeader("Content-Type", "application/json");
        try {
          const url = new URL(req.url || "/", "http://localhost");
          // ADR-0001: accept the production paths. `/public/*` and `/staff/*` map onto the
          // preview routes; only staff creation and staff cancellation keep their prefix.
          const staffRequest = url.pathname.startsWith("/staff/");
          if (url.pathname.startsWith("/public/")) url.pathname = url.pathname.slice("/public".length);
          else if (
            staffRequest &&
            !(url.pathname === "/staff/bookings" && req.method === "POST") &&
            !/^\/staff\/bookings\/[^/]+\/cancel$/.test(url.pathname)
          )
            url.pathname = url.pathname.slice("/staff".length);
          const role = (url.searchParams.get("role") || "PA") as Role;
          const s = session(role);
          const requestTime = now();
          const toStaffBooking = (booking: StoredBooking, actor: Session, arrival = false) =>
            projectStaff(
              {
                ...booking,
                attachmentAvailable: !!booking.attachmentToken && documents.has(booking.attachmentToken),
              },
              actor,
              { arrival, now: requestTime, allowedActions: allowedActions(actor, booking, requestTime) },
            );
          const toVisitorBooking = (booking: Booking) => projectVisitor(booking, requestTime);
          if (url.pathname === "/attachments" && req.method === "POST") {
            // ADR-0001: the production upload — multipart `file`, `reference` and `purpose`.
            const reply = (status: number, data: unknown) =>
              res.writeHead(status).end(JSON.stringify(data));
            const headers = new Headers();
            for (const [key, value] of Object.entries(req.headers))
              if (value !== undefined) headers.set(key, String(value));
            let form: FormData;
            try {
              form = await new Request("http://preview.local/attachments", {
                method: "POST",
                headers,
                body: Readable.toWeb(req) as ReadableStream,
                duplex: "half",
              } as RequestInit).formData();
            } catch {
              reply(422, { message: "Unggahan harus berupa formulir berkas." });
              return;
            }
            const file = form.get("file");
            const purpose = form.get("purpose");
            if (
              !(file instanceof File) ||
              typeof form.get("reference") !== "string" ||
              (purpose !== "document" && purpose !== "visitorPhoto")
            ) {
              reply(422, { message: "Periksa berkas yang diunggah." });
              return;
            }
            const photo = purpose === "visitorPhoto";
            const invalidType = photo ? "Gunakan foto JPEG atau PNG." : "Gunakan berkas PDF.";
            if (file.size > 5 * 1024 * 1024) {
              reply(413, { message: "Berkas maksimal 5 MB." });
              return;
            }
            const bytes = Buffer.from(await file.arrayBuffer());
            if (
              (photo ? !visitorPhotoTypes.includes(file.type) : file.type !== notaDinasType) ||
              !matchesAttachmentSignature(file.type, bytes)
            ) {
              reply(422, { message: invalidType });
              return;
            }
            const token = crypto.randomUUID();
            (photo ? visitorPhotos : documents).set(token, { bytes, type: file.type });
            reply(201, {
              token,
              expiresAt: new Date(now() + 15 * 60_000).toISOString(),
              serverTime: now(),
            });
            return;
          }
          let body: any = {};
          if (req.method === "POST") {
            let raw = "";
            for await (const chunk of req) {
              raw += chunk;
              if (raw.length > 100000) throw Error("Payload terlalu besar.");
            }
            body = JSON.parse(raw || "{}");
          }
          const send = (data: unknown, status = 200) => {
            res.statusCode = status;
            res.end(JSON.stringify(data));
          };
          if (!roles.includes(role)) {
            send({ message: "Peran tidak tersedia." }, 403);
            return;
          }
          // ADR-0001: attendance and room identity use the production wire format.
          const attendance = url.pathname.match(/^\/bookings\/([^/]+)\/attendance$/);
          if (attendance && req.method === "POST") {
            const direction = ({ check_in: "checkin", check_out: "checkout" } as const)[
              body.action as "check_in" | "check_out"
            ];
            if (!direction) {
              send(
                { message: "Periksa tindakan kehadiran.", fieldErrors: { action: ["Pilih check in atau check out."] } },
                422,
              );
              return;
            }
            url.pathname = `/bookings/${attendance[1]}/${direction}`;
            delete body.action;
          }
          if (req.method === "POST" && Object.hasOwn(body, "room")) {
            send({ message: "Gunakan roomId.", fieldErrors: { room: ["Gunakan roomId."] } }, 422);
            return;
          }
          if (req.method === "POST" && Object.hasOwn(body, "roomId")) {
            if (body.roomId === null || body.roomId === "") body.room = "";
            else {
              const room = roomOptions.find(({ id }) => id === body.roomId);
              if (!room) {
                send(
                  { message: "Pilih ruangan yang aktif.", fieldErrors: { roomId: ["Pilih ruangan yang aktif."] } },
                  422,
                );
                return;
              }
              body.room = room.name;
            }
            delete body.roomId;
          }
          const fileMatch = url.pathname.match(/^\/bookings\/([^/]+)\/(visitor-photo|attachment)$/);
          if (fileMatch && ["GET", "POST"].includes(req.method || "")) {
            const [, id, kind] = fileMatch;
            const credential =
              typeof body.credential === "string" ? body.credential.replace(/^VMS-DEMO:/, "") : "";
            const booking = scope(bookings, s).find((b) =>
              b.id === id &&
              !isRestricted(b) &&
              (s.role === "Receptionist"
                ? req.method === "POST" &&
                  ((b.status === "approved" && s.permissions.includes("checkin")) ||
                    (b.status === "checked_in" && s.permissions.includes("checkout"))) &&
                  b.code === credential
                : req.method === "GET"),
            );
            const token = kind === "visitor-photo" ? booking?.visitorPhotoToken : booking?.attachmentToken;
            const file = token && (kind === "visitor-photo" ? visitorPhotos : documents).get(token);
            if (!file) {
              send({ message: "Berkas tidak tersedia." }, 404);
              return;
            }
            res.setHeader("Content-Type", file.type);
            res.setHeader("X-Content-Type-Options", "nosniff");
            res.end(file.bytes);
            return;
          }
          if (url.pathname === "/session") {
            send(s);
            return;
          }
          if (["/reports/visits", "/reports/visits/export"].includes(url.pathname) && req.method === "GET") {
            const exporting = url.pathname.endsWith("/export");
            if (!canReadVisitLog(s) || (exporting && !canExportSummary(s))) {
              send({ message: "Akses log kunjungan tidak diizinkan." }, 403);
              return;
            }
            const values = Object.fromEntries(
              [...new Set(url.searchParams.keys())]
                .filter((key) => !["role", "page", "pageSize", "revision"].includes(key))
                .map((key) => [key, readFilter(url.searchParams, key, "")]),
            );
            const parsed = visitLogFilterSchema.safeParse(values);
            const page = Number(url.searchParams.get("page") || 1);
            const pageSize = Number(url.searchParams.get("pageSize") || 10);
            if (!parsed.success || !Number.isInteger(page) || page < 1 || page > 100000 || ![10, 25, 50].includes(pageSize)) {
              send({ message: "Periksa filter log kunjungan.", ...(!parsed.success ? { fieldErrors: parsed.error.flatten().fieldErrors } : {}) }, 422);
              return;
            }
            const all = projectVisitLog(bookings, requestTime);
            const items = selectVisitLog(all, parsed.data);
            const summary = summarizeVisitLog(items);
            const revision = createHash("sha256").update(JSON.stringify({ filter: parsed.data, items, permissions: [...s.permissions].sort() })).digest("hex");
            const snapshot = { filter: parsed.data, items, summary, revision, generatedAt: new Date(requestTime).toISOString() };
            if (exporting) {
              if (url.searchParams.get("revision") !== revision) {
                send({ message: "Log telah berubah. Perbarui sebelum mengekspor." }, 409);
                return;
              }
              send(snapshot);
            } else {
              const activePage = Math.min(page, Math.max(1, Math.ceil(items.length / pageSize)));
              send({ ...snapshot, items: items.slice((activePage - 1) * pageSize, activePage * pageSize),
                page: activePage, pageSize, total: items.length, canExport: canExportSummary(s),
                officers: [...new Map(all.filter((b) => b.officerId && b.host).map((b) => [b.officerId, { id: b.officerId, name: b.host }])).values()].sort((a, b) => a.name.localeCompare(b.name, "id")),
                rooms: [...new Set(all.map((b) => b.room).filter(Boolean))].sort(),
              });
            }
            return;
          }
          if (
            ["/reports/summary", "/reports/export"].includes(url.pathname) &&
            req.method === "GET"
          ) {
            const exporting = url.pathname.endsWith("/export");
            if (!(exporting ? canExportSummary(s) : canReadSummary(s))) {
              send({ message: "Akses laporan tidak diizinkan." }, 403);
              return;
            }
            const parsed = summaryFilterSchema.safeParse({
              from: url.searchParams.get("from"),
              to: url.searchParams.get("to"),
              status: readFilter(url.searchParams, "status", "all") || "all",
            });
            if (!parsed.success) {
              send(
                {
                  message: "Periksa filter laporan.",
                  fieldErrors: parsed.error.flatten().fieldErrors,
                },
                422,
              );
              return;
            }
            const result = aggregateVisits(bookings, s, parsed.data, requestTime);
            const revision = createHash("sha256")
              .update(
                JSON.stringify({
                  filter: result.filter,
                  scope: result.scope,
                  counts: result.counts,
                  role: s.role,
                  assignedOfficerIds: [...s.assignedOfficerIds].sort(),
                }),
              )
              .digest("hex");
            const summary: VisitSummary = { ...result, revision };
            if (exporting) {
              if (url.searchParams.get("revision") !== revision) {
                send(
                  { message: "Ringkasan telah berubah. Perbarui laporan sebelum mengekspor." },
                  409,
                );
                return;
              }
              send({
                filename: `kunjungan-ringkasan-${result.filter.from}-${result.filter.to}.csv`,
                csv: summaryCsv(summary),
                revision,
              });
            } else send(summary);
            return;
          }
          if (url.pathname === "/bookings" && req.method === "GET") {
            send(scope(bookings, s).map((b) => toStaffBooking(b, s)));
            return;
          }
          if (url.pathname === "/staff/bookings" && req.method === "POST") {
            const kind = body.visitKind === "restricted" ? "restricted" : "standard";
            if (!canCreateAppointment(s, undefined, kind)) {
              send({ message: "Peran ini tidak dapat membuat janji temu." }, 403);
              return;
            }
            const { idempotencyKey: submissionKey, ...restrictedInput } = body;
            const parsed =
              kind === "restricted"
                ? restrictedAppointmentSchema.safeParse(restrictedInput)
                : standardStaffAppointmentInputSchema.safeParse(body);
            if (!parsed.success) {
              send(
                {
                  message: "Periksa data janji temu.",
                  fieldErrors: parsed.error.flatten().fieldErrors,
                },
                422,
              );
              return;
            }
            const input = parsed.data;
            if (!input.officerId) {
              send(
                {
                  message: "Periksa data janji temu.",
                  fieldErrors: { officerId: ["Pilih pejabat dari daftar yang tersedia."] },
                },
                422,
              );
              return;
            }
            if (!canCreateAppointment(s, input.officerId, kind)) {
              send({ message: "Tujuan kunjungan di luar kewenangan Anda." }, 403);
              return;
            }
            if (
              typeof body.idempotencyKey !== "string" ||
              !body.idempotencyKey.trim() ||
              body.idempotencyKey.length > 128
            ) {
              send({ message: "Kunci pengiriman wajib valid." }, 422);
              return;
            }
            const attemptKey = `${s.role}:${body.idempotencyKey}`;
            const payload = JSON.stringify(input);
            const previous = staffAttempts.get(attemptKey);
            if (previous) {
              if (previous.payload !== payload)
                send(
                  {
                    message: "Data pengiriman berubah. Gunakan kunci pengiriman baru.",
                  },
                  409,
                );
              else send(toStaffBooking(previous.booking, s), 201);
              return;
            }
            if (input.room && !rooms.includes(input.room)) {
              send(
                {
                  message: "Pilih ruangan yang tersedia.",
                  fieldErrors: { roomId: ["Pilih ruangan yang tersedia."] },
                },
                422,
              );
              return;
            }
            const officer = officers.find((o) => o.id === input.officerId);
            const booking: StoredBooking = {
              ...input,
              name: input.name || "",
              organization: input.organization || "",
              officerId: input.officerId || "",
              email: "email" in input ? input.email : "",
              phone: "phone" in input ? input.phone : "",
              purpose: "purpose" in input ? input.purpose : "",
              identityType: "identityType" in input ? input.identityType : "",
              visitors: "visitors" in input ? input.visitors : [],
              id: crypto.randomUUID(),
              code: String(88000100 + bookings.length),
              visitKind: kind,
              host: officer?.name || "",
              vip: officer?.vip || false,
              duration: "duration" in input ? input.duration : 60,
              room: input.room || "",
              status: "approved",
              version: 1,
              createdBy: { name: s.name, role: s.role },
              history: [
                {
                  at: new Date(now()).toISOString(),
                  action: "received",
                  message: "Janji temu dibuat oleh petugas.",
                },
                {
                  at: new Date(now()).toISOString(),
                  action: "approved",
                  message: "Janji temu otomatis disetujui saat dibuat oleh petugas.",
                },
              ],
            };
            if (hasReservationConflict(booking, bookings)) {
              send(reservationConflict, 409);
              return;
            }
            bookings = [booking, ...bookings];
            staffAttempts.set(attemptKey, { payload, booking });
            send(toStaffBooking(booking, s), 201);
            return;
          }
          if (url.pathname === "/bookings" && req.method === "POST") {
            if (body.visitKind && body.visitKind !== "standard") {
              send({ message: "Jenis kunjungan tidak tersedia." }, 403);
              return;
            }
            if (
              typeof body.idempotencyKey !== "string" ||
              !body.idempotencyKey.trim() ||
              body.idempotencyKey.length > 128
            ) {
              send({ message: "Kunci pengiriman wajib valid." }, 422);
              return;
            }
            const key = body.idempotencyKey;
            const previous = attempts.get(key);
            if (previous?.state === "closed") {
              send(
                {
                  message:
                    "Referensi pengiriman sudah ditutup. Tinjau data sebelum mengirim kembali.",
                },
                409,
              );
              return;
            }
            if (previous) {
              const parsed = publicBookingInputSchema.safeParse(body);
              if (!parsed.success || previous.payload !== JSON.stringify(parsed.data))
                send(
                  {
                    message:
                      "Data pengiriman berubah. Gunakan referensi baru setelah hasil sebelumnya dipastikan.",
                  },
                  409,
                );
              else send(toVisitorBooking(previous.booking), 201);
              return;
            }
            const fresh = publicBookingCreateSchema(requestTime).safeParse(body);
            if (!fresh.success) {
              send(
                {
                  message: "Periksa data permohonan.",
                  fieldErrors: fresh.error.flatten().fieldErrors,
                },
                422,
              );
              return;
            }
            const { ack, ...input } = fresh.data;
            const o = officers.find((o) => o.id === input.officerId)!;
            const b: StoredBooking = {
              ...input,
              id: crypto.randomUUID(),
              code: String(88000100 + bookings.length),
              host: o.name,
              vip: o.vip,
              duration: 60,
              room: "Belum ditetapkan",
              status: "pending",
              version: 1,
              rejectionReason: undefined,
              history: [
                {
                  at: new Date(requestTime).toISOString(),
                  action: "received",
                  message: "Permohonan diterima",
                },
              ],
            };
            bookings = [b, ...bookings];
            attempts.set(key, {
              state: "committed",
              payload: JSON.stringify(fresh.data),
              booking: b,
            });
            send(toVisitorBooking(b), 201);
            return;
          }
          if (url.pathname === "/submissions/resolve" && req.method === "POST") {
            const email = emailSchema.safeParse(body.email);
            if (
              !email.success ||
              typeof body.idempotencyKey !== "string" ||
              !body.idempotencyKey.trim() ||
              body.idempotencyKey.length > 128
            ) {
              send({ message: "Periksa email dan referensi pengiriman." }, 422);
              return;
            }
            const previous = attempts.get(body.idempotencyKey);
            if (previous?.state === "committed") {
              if (previous.booking.email !== email.data.toLowerCase()) {
                send(
                  {
                    message: "Email pengajuan sebelumnya tidak cocok. Periksa email dan coba lagi.",
                  },
                  401,
                );
                return;
              }
              send({ outcome: "committed", booking: toVisitorBooking(previous.booking) });
            } else {
              // Seal an absent attempt before allowing a replacement; late POSTs must fail.
              attempts.set(body.idempotencyKey, { state: "closed" });
              send({ outcome: "not_committed" });
            }
            return;
          }
          if (url.pathname === "/lookup") {
            const b = bookings.find(
              (b) =>
                !isRestricted(b) &&
                b.code === String(body.code || "").trim() &&
                b.email.toLowerCase() ===
                  String(body.email || "")
                    .trim()
                    .toLowerCase(),
            );
            if (!b) {
              send(
                {
                  message: "Email atau kode booking tidak cocok. Periksa keduanya dan coba lagi.",
                },
                401,
              );
              return;
            }
            send(toVisitorBooking(b));
            return;
          }
          if (url.pathname === "/resolve") {
            const code = body.credential?.replace(/^VMS-DEMO:/, "");
            const b = scope(bookings, s).find((b) => b.code === code);
            if (!b || (body.visitId && b.id !== body.visitId)) {
              send({ message: "Kunjungan tidak ditemukan. Periksa kode." }, 404);
              return;
            }
            send(toStaffBooking(b, s, true));
            return;
          }
          const detail = url.pathname.match(/^\/bookings\/([^/]+)$/);
          if (detail && req.method === "GET") {
            const b = s.role === "Receptionist"
              ? undefined
              : scope(bookings, s).find((b) => b.id === detail[1]);
            if (!b) send({ message: "Kunjungan tidak ditemukan." }, 404);
            else send(toStaffBooking(b, s));
            return;
          }
          const match = url.pathname.match(
            /^\/(?:staff\/)?bookings\/([^/]+)\/(approve|reject|reschedule|checkin|checkout|cancel|update)$/,
          );
          if (match) {
            const b = bookings.find((b) => b.id === match[1]);
            const action = match[2];
            if (action === "approve" || action === "reschedule") {
              for (const key of ["visitorInstructions", "coordinationNote"] as const) {
                const result = visitNoteFields[key].safeParse(body[key]);
                if (!result.success) {
                  send(
                    {
                      message: "Periksa catatan kunjungan.",
                      fieldErrors: { [key]: [result.error.issues[0].message] },
                    },
                    422,
                  );
                  return;
                }
              }
            }
            const staffCancellation = action === "cancel" && staffRequest;
            if (
              !b ||
              (staffCancellation && !scope([b], s).length) ||
              (isRestricted(b) &&
                ((action === "cancel" && !staffCancellation) || !scope([b], s).length))
            ) {
              send({ message: "Kunjungan tidak ditemukan." }, 404);
              return;
            }
            if (action === "update") {
              if (!can(s, b, "update", requestTime)) {
                send({ message: "Tindakan tidak tersedia." }, 403);
                return;
              }
              if (body.version !== b.version) {
                send({ message: "Data telah berubah. Muat ulang sebelum melanjutkan." }, 409);
                return;
              }
              const { version: previousVersion, ...restrictedInput } = body;
              const parsed = restrictedAppointmentSchema.safeParse(restrictedInput);
              if (!parsed.success) {
                send(
                  {
                    message: "Periksa data kunjungan.",
                    fieldErrors: parsed.error.flatten().fieldErrors,
                  },
                  422,
                );
                return;
              }
              const officerId = Object.hasOwn(body, "officerId")
                ? parsed.data.officerId || ""
                : b.officerId;
              if (!officerId) {
                send(
                  {
                    message: "Periksa data kunjungan.",
                    fieldErrors: { officerId: ["Pilih pejabat dari daftar yang tersedia."] },
                  },
                  422,
                );
                return;
              }
              if (!canCreateAppointment(s, officerId, "restricted")) {
                send({ message: "Tujuan tidak tersedia." }, 403);
                return;
              }
              // Schedule changes use the existing reschedule action and conflict checks.
              if (parsed.data.date !== b.date || parsed.data.time !== b.time) {
                send({ message: "Gunakan Ubah jadwal untuk mengganti tanggal atau waktu." }, 422);
                return;
              }
              const officer = officers.find((o) => o.id === officerId);
              if (
                b.status === "approved" &&
                hasReservationConflict({ ...b, officerId }, bookings, b.id)
              ) {
                send(reservationConflict, 409);
                return;
              }
              if (Object.hasOwn(body, "name")) b.name = parsed.data.name || "";
              if (Object.hasOwn(body, "organization"))
                b.organization = parsed.data.organization || "";
              if (Object.hasOwn(body, "serviceId")) {
                if (parsed.data.serviceId) b.serviceId = parsed.data.serviceId;
                else delete b.serviceId;
              }
              Object.assign(b, {
                officerId,
                ...(Object.hasOwn(body, "visitorInstructions")
                  ? { visitorInstructions: parsed.data.visitorInstructions }
                  : {}),
                ...(Object.hasOwn(body, "coordinationNote")
                  ? { coordinationNote: parsed.data.coordinationNote }
                  : {}),
                host: officer?.name || "",
                vip: officer?.vip || false,
              });
              b.version++;
              b.history.push({
                at: new Date(now()).toISOString(),
                action: "updated",
                message: "Protokol memperbarui detail kunjungan terbatas.",
              });
              send(toStaffBooking(b, s));
              return;
            }
            if (body.version !== b.version) {
              send(
                {
                  message: "Data telah berubah. Muat ulang sebelum melanjutkan.",
                },
                409,
              );
              return;
            }
            if (action === "cancel" && !staffCancellation) {
              if (
                body.code !== b.code ||
                String(body.email || "")
                  .trim()
                  .toLowerCase() !== b.email.toLowerCase() ||
                !["pending", "approved"].includes(b.status)
              ) {
                send({ message: "Pembatalan tidak tersedia." }, 403);
                return;
              }
              if (!canVisitorCancel(b, requestTime)) {
                send(
                  {
                    message:
                      "Pembatalan mandiri hanya tersedia sebelum waktu mulai kunjungan. Hubungi petugas untuk bantuan.",
                  },
                  409,
                );
                return;
              }
            } else if (
              action === "checkin" &&
              b.status === "approved" &&
              s.role === "Receptionist" &&
              !getVisitTiming(b, requestTime).canCheckIn
            ) {
              send(
                {
                  message:
                    getVisitTiming(b, requestTime).displayStatus === "missed"
                      ? "Waktu kedatangan telah berakhir. Kunjungan terlewat (Missed)."
                      : "Check-in dibuka 30 menit sebelum jadwal sampai waktu selesai kunjungan.",
                },
                409,
              );
              return;
            } else if (!can(s, b, action as any, requestTime)) {
              send(
                {
                  message: "Tindakan tidak diizinkan untuk peran atau status ini.",
                },
                403,
              );
              return;
            }
            if (staffCancellation) {
              const parsed = staffCancellationSchema.safeParse(body);
              if (!parsed.success) {
                send(
                  {
                    message: "Periksa alasan pembatalan.",
                    fieldErrors: parsed.error.flatten().fieldErrors,
                  },
                  422,
                );
                return;
              }
              body.reason = parsed.data.reason;
              b.cancellationReason = parsed.data.reason;
              b.internalCancellationNote = parsed.data.internalCancellationNote || undefined;
            }
            if (
              ["checkin", "checkout"].includes(action) &&
              body.credential?.replace(/^VMS-DEMO:/, "") !== b.code
            ) {
              send({ message: "Kunjungan tidak ditemukan." }, 404);
              return;
            }
            if (action === "approve" && body.room && !rooms.includes(body.room))
              throw Error("Pilih ruangan yang aktif.");
            if (
              action === "approve" &&
              (!Number.isInteger(body.duration ?? 60) ||
                (body.duration ?? 60) < 15 ||
                (body.duration ?? 60) > 240)
            )
              throw Error("Durasi harus 15–240 menit.");
            if (
              action === "approve" &&
              hasReservationConflict(
                { ...b, room: body.room ?? b.room, duration: body.duration ?? 60 },
                bookings,
                b.id,
              )
            ) {
              send(reservationConflict, 409);
              return;
            }
            if (action === "reject") {
              const parsed = rejectionSchema.safeParse(body);
              if (!parsed.success) {
                send(
                  {
                    message: "Periksa keterangan penolakan.",
                    fieldErrors: parsed.error.flatten().fieldErrors,
                  },
                  422,
                );
                return;
              }
              b.rejectionReason = parsed.data.rejectionReason;
              b.internalRejectionNote = parsed.data.internalRejectionNote || undefined;
            }
            if (action === "reschedule") {
              body.room ??= b.room;
              if (isRestricted(b)) {
                body.duration ??= b.duration;
                body.reason ??= "";
              }
              const error = validateReschedule(
                body.date,
                body.time,
                body.duration,
                body.room,
                body.reason,
                isRestricted(b),
                body.room ? rooms : null,
                requestTime,
              );
              if (error) throw Error(error);
              if (
                hasReservationConflict(
                  {
                    ...b,
                    date: body.date,
                    time: body.time,
                    duration: body.duration,
                    room: body.room,
                  },
                  bookings,
                  b.id,
                )
              ) {
                send(reservationConflict, 409);
                return;
              }
              Object.assign(b, {
                date: body.date,
                time: body.time,
                duration: body.duration,
                room: body.room,
                rescheduled: true,
              });
            } else if (action === "approve") {
              b.status = "approved";
              b.room = body.room ?? b.room;
              b.duration = body.duration || 60;
            } else
              b.status = (
                {
                  reject: "rejected",
                  checkin: "checked_in",
                  checkout: "checked_out",
                  cancel: "cancelled",
                } as const
              )[action as "reject"];
            if (action === "approve" || action === "reschedule") {
              for (const key of ["visitorInstructions", "coordinationNote"] as const) {
                if (Object.hasOwn(body, key)) b[key] = String(body[key]).trim();
              }
            }
            b.version++;
            const eventActions: Record<string, PublicVisitAction> = {
              approve: "approved",
              reject: "rejected",
              reschedule: "rescheduled",
              cancel: "cancelled",
              checkin: "checked_in",
              checkout: "checked_out",
            };
            b.history.push({
              at: new Date(now()).toISOString(),
              action: eventActions[action] || "updated",
              ...(action === "reschedule" ? { publicMessage: body.reason } : {}),
              message:
                action === "reject"
                  ? `Permohonan ditolak: ${b.rejectionReason}`
                  : action === "reschedule"
                    ? `Jadwal ditetapkan ulang: ${body.reason}`
                    : `${s.role}: ${action}${body.reason ? ` · ${body.reason}` : ""}`,
            });
            send(
              action === "cancel" && !staffCancellation
                ? toVisitorBooking(b)
                : toStaffBooking(b, s),
            );
            return;
          }
          send({ message: "Operasi preview tidak tersedia." }, 404);
        } catch (error) {
          res.statusCode = 422;
          res.end(
            JSON.stringify({
              message: error instanceof Error ? error.message : "Permintaan gagal.",
            }),
          );
        }
      });
    },
  };
}
