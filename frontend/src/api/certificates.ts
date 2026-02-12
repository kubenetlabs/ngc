import apiClient from "./client";
import type { Certificate } from "@/types/certificate";

export async function fetchCertificates(): Promise<Certificate[]> {
  const { data } = await apiClient.get<Certificate[]>("/certificates");
  return data;
}

export async function fetchExpiringCertificates(days?: number): Promise<Certificate[]> {
  const params = days ? { days } : {};
  const { data } = await apiClient.get<Certificate[]>("/certificates/expiring", { params });
  return data;
}

export async function deleteCertificate(name: string): Promise<void> {
  await apiClient.delete(`/certificates/${name}`);
}
