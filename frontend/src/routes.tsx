import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { MainLayout } from "@/components/layout/MainLayout";

const Dashboard = lazy(() => import("@/pages/Dashboard"));
const GatewayList = lazy(() => import("@/pages/GatewayList"));
const GatewayCreate = lazy(() => import("@/pages/GatewayCreate"));
const GatewayDetail = lazy(() => import("@/pages/GatewayDetail"));
const GatewayEdit = lazy(() => import("@/pages/GatewayEdit"));
const InferenceOverview = lazy(() => import("@/pages/InferenceOverview"));
const InferencePoolList = lazy(() => import("@/pages/InferencePoolList"));
const InferencePoolCreate = lazy(() => import("@/pages/InferencePoolCreate"));
const InferencePoolDetail = lazy(() => import("@/pages/InferencePoolDetail"));
const RouteList = lazy(() => import("@/pages/RouteList"));
const RouteCreate = lazy(() => import("@/pages/RouteCreate"));
const RouteDetail = lazy(() => import("@/pages/RouteDetail"));
const RouteEdit = lazy(() => import("@/pages/RouteEdit"));
const PolicyList = lazy(() => import("@/pages/PolicyList"));
const PolicyCreate = lazy(() => import("@/pages/PolicyCreate"));
const CertificateList = lazy(() => import("@/pages/CertificateList"));
const ObservabilityDashboard = lazy(() => import("@/pages/ObservabilityDashboard"));
const LogExplorer = lazy(() => import("@/pages/LogExplorer"));
const DiagnosticsHome = lazy(() => import("@/pages/DiagnosticsHome"));
const RouteCheck = lazy(() => import("@/pages/RouteCheck"));
const XCOverview = lazy(() => import("@/pages/XCOverview"));
const CoexistenceDashboard = lazy(() => import("@/pages/CoexistenceDashboard"));
const MigrationList = lazy(() => import("@/pages/MigrationList"));
const MigrationNew = lazy(() => import("@/pages/MigrationNew"));
const AuditLog = lazy(() => import("@/pages/AuditLog"));
const SettingsPage = lazy(() => import("@/pages/SettingsPage"));
const ClusterManagement = lazy(() => import("@/pages/ClusterManagement"));
const ClusterDetail = lazy(() => import("@/pages/ClusterDetail"));
const ClusterRegister = lazy(() => import("@/pages/ClusterRegister"));

export const routes: RouteObject[] = [
  {
    element: <MainLayout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: "clusters", element: <ClusterManagement /> },
      { path: "clusters/register", element: <ClusterRegister /> },
      { path: "clusters/:name", element: <ClusterDetail /> },
      { path: "gateways", element: <GatewayList /> },
      { path: "gateways/create", element: <GatewayCreate /> },
      { path: "gateways/:ns/:name", element: <GatewayDetail /> },
      { path: "gateways/:ns/:name/edit", element: <GatewayEdit /> },
      { path: "inference", element: <InferenceOverview /> },
      { path: "inference/pools", element: <InferencePoolList /> },
      { path: "inference/pools/create", element: <InferencePoolCreate /> },
      { path: "inference/pools/:ns/:name", element: <InferencePoolDetail /> },
      { path: "routes", element: <RouteList /> },
      { path: "routes/create/:type", element: <RouteCreate /> },
      { path: "routes/:ns/:name", element: <RouteDetail /> },
      { path: "routes/:ns/:name/edit", element: <RouteEdit /> },
      { path: "policies", element: <PolicyList /> },
      { path: "policies/create", element: <PolicyCreate /> },
      { path: "policies/create/:type", element: <PolicyCreate /> },
      { path: "certificates", element: <CertificateList /> },
      { path: "observability", element: <ObservabilityDashboard /> },
      { path: "observability/logs", element: <LogExplorer /> },
      { path: "diagnostics", element: <DiagnosticsHome /> },
      { path: "diagnostics/route-check", element: <RouteCheck /> },
      { path: "xc", element: <XCOverview /> },
      { path: "coexistence", element: <CoexistenceDashboard /> },
      { path: "migration", element: <MigrationList /> },
      { path: "migration/new", element: <MigrationNew /> },
      { path: "audit", element: <AuditLog /> },
      { path: "settings", element: <SettingsPage /> },
      {
        path: "*",
        element: (
          <div className="flex flex-col items-center justify-center py-20">
            <h1 className="text-4xl font-bold text-foreground">404</h1>
            <p className="mt-2 text-muted-foreground">Page not found</p>
            <a href="/" className="mt-4 text-sm text-blue-400 hover:underline">
              Back to Dashboard
            </a>
          </div>
        ),
      },
    ],
  },
];
