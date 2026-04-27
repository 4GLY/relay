import { redirect } from "next/navigation";

type Params = Promise<{ snapshotId: string }>;

export default async function SnapshotPage({ params }: { params: Params }) {
  const { snapshotId } = await params;
  redirect(`/p/${encodeURIComponent(snapshotId)}`);
}
