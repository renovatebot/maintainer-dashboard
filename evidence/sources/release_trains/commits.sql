-- Renovate 44 was an accidental major bump of the regular 43 train, so it is folded into 43
select
    *,
    case when major_version = 44 then 43 else major_version end as release_train
from commits;
