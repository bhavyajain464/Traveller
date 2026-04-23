import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

function BillsPage() {
  const [form, setForm] = useState({
    user_id: "demo-user-id",
    bill_id: "demo-bill-id",
    date: "",
    payment_id: "pay-demo-001",
    payment_method: "upi"
  });
  const [data, setData] = useState(null);
  const [error, setError] = useState("");

  const set = (key, value) => setForm((s) => ({ ...s, [key]: value }));
  const run = async (req) => {
    try {
      setError("");
      setData(await req());
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <section>
      <h2>Daily Bills API</h2>
      <div className="card-grid">
        {Object.entries(form).map(([key, value]) => (
          <label key={key} className="field">
            <span>{key}</span>
            <input value={value} onChange={(e) => set(key, e.target.value)} />
          </label>
        ))}
      </div>
      <div className="toolbar">
        <button onClick={() => run(() => apiRequest(`/bills/users/${form.user_id}${form.date ? `?date=${form.date}` : ""}`))}>GET /bills/users/:user_id</button>
        <button onClick={() => run(() => apiRequest(`/bills/users/${form.user_id}/pending`))}>GET /bills/users/:user_id/pending</button>
        <button onClick={() => run(() => apiRequest(`/bills/${form.bill_id}/pay`, { method: "POST", body: JSON.stringify({ payment_id: form.payment_id, payment_method: form.payment_method }) }))}>POST /bills/:bill_id/pay</button>
        <button onClick={() => run(() => apiRequest(`/bills/generate${form.date ? `?date=${form.date}` : ""}`, { method: "POST" }))}>POST /bills/generate</button>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default BillsPage;
