import {redirect} from 'next/navigation';

export default function NodesPage({params}: {params: {locale: string}}) {
  redirect(`/${params.locale}/nodes/connect`);
}
