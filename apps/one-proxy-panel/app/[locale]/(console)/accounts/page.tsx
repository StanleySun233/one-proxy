import {redirect} from 'next/navigation';

export default function AccountsPage({params}: {params: {locale: string}}) {
  redirect(`/${params.locale}/accounts/create`);
}
