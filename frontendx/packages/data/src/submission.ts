import type { PublicBooking } from "./domain.ts";
import { isDefinitiveRejection, isTemporaryNetworkError } from "./errors.ts";
import { publicBookingInputSchema, type PublicBookingInput } from "./validation.ts";

export type SubmissionResolution =
  { outcome: "committed"; booking: PublicBooking } | { outcome: "not_committed" };
export type SubmissionPhase =
  "idle" | "sending" | "retrying" | "uncertain" | "checking" | "received";
export interface SubmissionStore {
  read(): string | null;
  write(key: string): void;
  clear(): void;
}
interface SubmissionApi {
  create(input: PublicBookingInput, key: string): Promise<PublicBooking>;
  resolveSubmission(key: string, email: string): Promise<SubmissionResolution>;
}
type Listener = (phase: SubmissionPhase) => void;

export class BookingSubmission {
  phase: SubmissionPhase;
  private key: string | null;
  private api: SubmissionApi;
  private store: SubmissionStore;
  constructor(api: SubmissionApi, store: SubmissionStore) {
    this.api = api;
    this.store = store;
    this.key = store.read();
    this.phase = this.key ? "uncertain" : "idle";
  }
  private change(phase: SubmissionPhase, notify: Listener) {
    this.phase = phase;
    notify(phase);
  }
  get canSubmit() {
    return this.phase === "idle";
  }
  private clear() {
    if (this.store.read() === this.key) this.store.clear();
    this.key = null;
  }
  confirmReceipt(notify: Listener) {
    if (this.phase !== "received") return;
    this.clear();
    this.change("idle", notify);
  }
  async submit(input: PublicBookingInput, notify: Listener): Promise<PublicBooking> {
    if (!this.canSubmit)
      throw new Error("Periksa hasil pengiriman sebelumnya sebelum mengirim kembali.");
    const pendingKey = this.store.read();
    if (pendingKey) {
      this.key = pendingKey;
      this.change("uncertain", notify);
      throw new Error("Periksa hasil pengiriman sebelumnya sebelum mengirim kembali.");
    }
    const normalized = publicBookingInputSchema.parse(input);
    const key = crypto.randomUUID();
    this.store.write(key);
    this.key = key;
    this.change("sending", notify);
    let uncertainFirstAttempt = false;
    try {
      let booking: PublicBooking;
      try {
        booking = await this.api.create(normalized, key);
      } catch (error) {
        if (!isTemporaryNetworkError(error)) throw error;
        uncertainFirstAttempt = true;
        this.change("retrying", notify);
        booking = await this.api.create(normalized, key);
      }
      this.change("received", notify);
      return booking;
    } catch (error) {
      if (!uncertainFirstAttempt && isDefinitiveRejection(error)) {
        this.clear();
        this.change("idle", notify);
      } else this.change("uncertain", notify);
      throw error;
    }
  }
  async recover(email: string, notify: Listener): Promise<SubmissionResolution> {
    if (this.phase !== "uncertain" || !this.key)
      throw new Error("Hasil pengiriman belum dapat diperiksa.");
    this.change("checking", notify);
    try {
      const result = await this.api.resolveSubmission(this.key, email);
      if (result.outcome === "committed") this.change("received", notify);
      else {
        this.clear();
        this.change("idle", notify);
      }
      return result;
    } catch (error) {
      this.change("uncertain", notify);
      throw error;
    }
  }
}
