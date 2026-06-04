import {redirect} from 'next/navigation';

export default async function AuditIndexPage({params}: {params: Promise<{locale: string}>}) {
  const {locale} = await params;
  redirect(`/${locale}/audit/dashboard`);
}
