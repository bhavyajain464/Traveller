import { Navigate, Route, Routes } from "react-router-dom";
import Layout from "./components/Layout";
import ProtectedRoute from "./components/ProtectedRoute";
import BillsPage from "./pages/BillsPage";
import BoardingsPage from "./pages/BoardingsPage";
import DashboardPage from "./pages/DashboardPage";
import FaresPage from "./pages/FaresPage";
import JourneysPage from "./pages/JourneysPage";
import LoginPage from "./pages/LoginPage";
import ProfilePage from "./pages/ProfilePage";
import RealtimePage from "./pages/RealtimePage";
import RoutesPage from "./pages/RoutesPage";
import SessionsPage from "./pages/SessionsPage";
import StopsPage from "./pages/StopsPage";
import TicketsPage from "./pages/TicketsPage";
import VehiclesPage from "./pages/VehiclesPage";
import "./App.css";

function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={(
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        )}
      >
        <Route path="/" element={<DashboardPage />} />
        <Route path="/plan" element={<JourneysPage />} />
        <Route path="/departures" element={<StopsPage />} />
        <Route path="/tickets" element={<TicketsPage />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/stops" element={<StopsPage />} />
        <Route path="/routes" element={<RoutesPage />} />
        <Route path="/routes/:id" element={<RoutesPage />} />
        <Route path="/journeys" element={<JourneysPage />} />
        <Route path="/sessions" element={<SessionsPage />} />
        <Route path="/boardings" element={<BoardingsPage />} />
        <Route path="/fares" element={<FaresPage />} />
        <Route path="/realtime" element={<RealtimePage />} />
        <Route path="/vehicles" element={<VehiclesPage />} />
        <Route path="/bills" element={<BillsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default App;
