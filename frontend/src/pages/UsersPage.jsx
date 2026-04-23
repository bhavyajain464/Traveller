import { useState } from "react";
import ApiResult from "../components/ApiResult";
import { apiRequest } from "../lib/api";

function UsersPage() {
  const [form, setForm] = useState({
    id: "demo-user-id",
    phone: "9999999999",
    name: "Demo User",
    email: "demo@traveller.app"
  });
  const [data, setData] = useState(null);
  const [error, setError] = useState("");

  const run = async (fn) => {
    try {
      setError("");
      const result = await fn();
      setData(result);
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <section>
      <h2>Users API</h2>
      <div className="card-grid">
        <article className="card">
          <h3>Create User</h3>
          <input placeholder="Phone number" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          <input placeholder="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input placeholder="Email (optional)" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          <button onClick={() => run(() => apiRequest("/users", { method: "POST", body: JSON.stringify({ phone_number: form.phone, name: form.name, email: form.email || undefined }) }))}>POST /users</button>
        </article>
        <article className="card">
          <h3>Get User</h3>
          <input placeholder="User ID" value={form.id} onChange={(e) => setForm({ ...form, id: e.target.value })} />
          <button onClick={() => run(() => apiRequest(`/users/${form.id}`))}>GET /users/:id</button>
          <h3>Delete by Phone</h3>
          <button onClick={() => run(() => apiRequest(`/users/phone/${form.phone}`, { method: "DELETE" }))}>DELETE /users/phone/:phone</button>
        </article>
      </div>
      <ApiResult data={data} error={error} />
    </section>
  );
}

export default UsersPage;
