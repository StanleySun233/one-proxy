import {redirect} from 'next/navigation';

export default function HealthPage({params}: {params: {locale: string}}) {
  redirect(`/${params.locale}/health/overview`);
}
