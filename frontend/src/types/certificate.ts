export interface Certificate {
  name: string;
  namespace: string;
  domains: string[];
  issuer: string;
  notBefore: string;
  notAfter: string;
  daysLeft: number;
  createdAt: string;
}
